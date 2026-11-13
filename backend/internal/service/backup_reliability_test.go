//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type failingBackupRepo struct {
	*mockSettingRepo
	readKey  string
	writeKey string
}

func (r *failingBackupRepo) GetValue(ctx context.Context, key string) (string, error) {
	if key == r.readKey {
		return "", errors.New("storage unavailable")
	}
	return r.mockSettingRepo.GetValue(ctx, key)
}
func (r *failingBackupRepo) Set(ctx context.Context, key, value string) error {
	if key == r.writeKey {
		return errors.New("storage unavailable")
	}
	return r.mockSettingRepo.Set(ctx, key, value)
}

type failingDeleteStore struct{ *mockObjectStore }

func (s *failingDeleteStore) Delete(context.Context, string) error {
	return errors.New("S3 unavailable")
}

func TestBackupReliability_Retention(t *testing.T) {
	ctx := context.Background()
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	store := newMockObjectStore()
	svc := newTestBackupService(repo, &mockDumper{dumpData: []byte("data")}, store)
	now := time.Now()
	for i, record := range []BackupRecord{
		{ID: "failed", Status: "failed", TriggeredBy: "scheduled"},
		{ID: "running", Status: "running", TriggeredBy: "scheduled"},
		{ID: "manual", Status: "completed", TriggeredBy: "manual"},
		{ID: "newest", Status: "completed", TriggeredBy: "scheduled"},
		{ID: "older", Status: "completed", TriggeredBy: "scheduled"},
		{ID: "expired-manual", Status: "completed", TriggeredBy: "manual", ExpiresAt: now.Add(-time.Hour).Format(time.RFC3339)},
		{ID: "restoring", Status: "completed", TriggeredBy: "scheduled", RestoreStatus: "running"},
	} {
		record.StartedAt = now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339)
		record.S3Key = record.ID
		store.objects[record.ID] = []byte("backup")
		require.NoError(t, svc.saveRecord(ctx, &record))
	}
	require.NoError(t, svc.cleanupOldBackups(ctx, &BackupScheduleConfig{RetainCount: 1, RetainDays: 1}))
	for _, id := range []string{"failed", "running", "manual", "newest", "restoring"} {
		_, err := svc.GetBackupRecord(ctx, id)
		require.NoError(t, err, id)
	}
	for _, id := range []string{"older", "expired-manual"} {
		_, err := svc.GetBackupRecord(ctx, id)
		require.ErrorIs(t, err, ErrBackupNotFound)
		require.NotContains(t, store.objects, id)
	}
}

func TestBackupReliability_ScheduledZeroExpiry(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	require.NoError(t, repo.Set(context.Background(), settingKeyBackupSchedule, `{"enabled":true,"retain_days":0,"retain_count":0}`))
	svc := newTestBackupService(repo, &mockDumper{dumpData: []byte("data")}, newMockObjectStore())
	svc.runScheduledBackup()
	records, err := svc.ListBackups(context.Background())
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Empty(t, records[0].ExpiresAt)
}

func TestBackupReliability_DeleteFailurePreservesRecord(t *testing.T) {
	ctx := context.Background()
	for _, cleanup := range []bool{false, true} {
		t.Run(fmt.Sprint(cleanup), func(t *testing.T) {
			repo := newMockSettingRepo()
			seedS3Config(t, repo)
			svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
			svc.store = &failingDeleteStore{newMockObjectStore()}
			svc.s3Cfg = &BackupS3Config{}
			require.NoError(t, svc.saveRecord(ctx, &BackupRecord{ID: "keep", Status: "completed", S3Key: "keep", ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339)}))
			var err error
			if cleanup {
				err = svc.cleanupOldBackups(ctx, &BackupScheduleConfig{})
			} else {
				err = svc.DeleteBackup(ctx, "keep")
			}
			require.ErrorContains(t, err, "S3 unavailable")
			_, err = svc.GetBackupRecord(ctx, "keep")
			require.NoError(t, err)
		})
	}
}

func TestBackupReliability_CorruptHistoryIsNotOverwritten(t *testing.T) {
	repo := newMockSettingRepo()
	require.NoError(t, repo.Set(context.Background(), settingKeyBackupRecords, "corrupt"))
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	require.ErrorIs(t, svc.saveRecord(context.Background(), &BackupRecord{ID: "new"}), ErrBackupRecordsCorrupt)
	raw, _ := repo.GetValue(context.Background(), settingKeyBackupRecords)
	require.Equal(t, "corrupt", raw)
}

func TestBackupReliability_ReadFailureIsNotAbsence(t *testing.T) {
	for _, key := range []string{settingKeyBackupRecords, settingKeyBackupS3Config, settingKeyBackupSchedule} {
		t.Run(key, func(t *testing.T) {
			repo := newMockSettingRepo()
			svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
			svc.settingRepo = &failingBackupRepo{mockSettingRepo: repo, readKey: key}
			var err error
			switch key {
			case settingKeyBackupRecords:
				err = svc.saveRecord(context.Background(), &BackupRecord{ID: "new"})
			case settingKeyBackupS3Config:
				_, err = svc.GetS3Config(context.Background())
			case settingKeyBackupSchedule:
				_, err = svc.GetSchedule(context.Background())
			}
			require.ErrorContains(t, err, "storage unavailable")
			require.Empty(t, repo.data)
		})
	}
}

func TestBackupReliability_RetainsAllSuccessfulRecords(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	for i := 0; i < 105; i++ {
		require.NoError(t, svc.saveRecord(context.Background(), &BackupRecord{ID: fmt.Sprint(i), Status: "completed"}))
	}
	records, err := svc.ListBackups(context.Background())
	require.NoError(t, err)
	require.Len(t, records, 105)
}

func TestBackupReliability_OperationsMutuallyExclusive(t *testing.T) {
	svc := newTestBackupService(newMockSettingRepo(), &mockDumper{}, newMockObjectStore())
	for _, restoring := range []bool{false, true} {
		require.NoError(t, svc.beginOperation(restoring))
		expected := ErrBackupInProgress
		if restoring {
			expected = ErrRestoreInProgress
		}
		_, err := svc.StartBackup(context.Background(), "manual", 0)
		require.ErrorIs(t, err, expected)
		_, err = svc.CreateBackup(context.Background(), "manual", 0)
		require.ErrorIs(t, err, expected)
		_, err = svc.StartRestore(context.Background(), "id")
		require.ErrorIs(t, err, expected)
		require.ErrorIs(t, svc.RestoreBackup(context.Background(), "id"), expected)
		require.ErrorIs(t, svc.DeleteBackup(context.Background(), "id"), expected)
		require.ErrorIs(t, svc.cleanupOldBackups(context.Background(), &BackupScheduleConfig{}), expected)
		svc.finishOperation(restoring)
	}
	svc.Stop()
	require.Error(t, svc.beginOperation(false))
	require.Error(t, svc.cleanupOldBackups(context.Background(), &BackupScheduleConfig{}))
}

func TestBackupReliability_RestoreMustPersistBeforeLaunching(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	dumper := &mockDumper{dumpData: []byte("data")}
	svc := newTestBackupService(repo, dumper, newMockObjectStore())
	record, err := svc.CreateBackup(context.Background(), "manual", 0)
	require.NoError(t, err)
	svc.settingRepo = &failingBackupRepo{mockSettingRepo: repo, writeKey: settingKeyBackupRecords}
	_, err = svc.StartRestore(context.Background(), record.ID)
	require.ErrorContains(t, err, "save initial restore record")
	svc.wg.Wait()
	require.Empty(t, dumper.restored)
	require.False(t, svc.restoring)
}

type closeErrorReader struct{ io.Reader }

func (closeErrorReader) Close() error { return errors.New("pg_dump exit failure") }

type closeErrorDumper struct{ mockDumper }

func (*closeErrorDumper) Dump(context.Context) (io.ReadCloser, error) {
	return closeErrorReader{strings.NewReader("partial SQL")}, nil
}

func TestBackupReliability_DumpExitFailureCannotSucceed(t *testing.T) {
	for _, async := range []bool{false, true} {
		t.Run(fmt.Sprint(async), func(t *testing.T) {
			repo := newMockSettingRepo()
			seedS3Config(t, repo)
			svc := newTestBackupService(repo, &closeErrorDumper{}, newMockObjectStore())
			var record *BackupRecord
			var err error
			if async {
				record, err = svc.StartBackup(context.Background(), "manual", 0)
				require.NoError(t, err)
				svc.wg.Wait()
			} else {
				record, err = svc.CreateBackup(context.Background(), "manual", 0)
				require.Error(t, err)
			}
			stored, err := svc.GetBackupRecord(context.Background(), record.ID)
			require.NoError(t, err)
			require.Equal(t, "failed", stored.Status)
			require.Contains(t, stored.ErrorMsg, "pg_dump exit failure")
		})
	}
}

func TestBackupReliability_ExpiryValidationAndUniqueObjects(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	store := newMockObjectStore()
	svc := newTestBackupService(repo, &mockDumper{dumpData: []byte("data")}, store)
	for _, days := range []int{-1, 36501} {
		_, err := svc.StartBackup(context.Background(), "manual", days)
		require.Error(t, err)
		_, err = svc.CreateBackup(context.Background(), "manual", days)
		require.Error(t, err)
		_, err = svc.UpdateSchedule(context.Background(), BackupScheduleConfig{RetainDays: days})
		require.Error(t, err)
	}
	first, err := svc.CreateBackup(context.Background(), "manual", 0)
	require.NoError(t, err)
	second, err := svc.CreateBackup(context.Background(), "manual", 0)
	require.NoError(t, err)
	require.NotEqual(t, first.S3Key, second.S3Key)
	require.Len(t, store.objects, 2)
}

// The real setting repository returns ErrSettingNotFound for a fresh install.
type missingBackupSettings struct{ *mockSettingRepo }

func (r *missingBackupSettings) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}
func TestBackupReliability_FreshInstall(t *testing.T) {
	repo := newMockSettingRepo()
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	svc.settingRepo = &missingBackupSettings{repo}
	ctx := context.Background()
	cfg, err := svc.GetSchedule(ctx)
	require.NoError(t, err)
	require.Equal(t, 14, cfg.RetainDays)
	_, err = svc.GetS3Config(ctx)
	require.NoError(t, err)
	records, err := svc.ListBackups(ctx)
	require.NoError(t, err)
	require.Empty(t, records)
	require.NoError(t, svc.saveRecord(ctx, &BackupRecord{ID: "first", Status: "completed"}))
}

func TestBackupReliability_PreservesSecretEncryptionOnConfigEdit(t *testing.T) {
	repo := newMockSettingRepo()
	seedS3Config(t, repo)
	svc := newTestBackupService(repo, &mockDumper{}, newMockObjectStore())
	_, err := svc.UpdateS3Config(context.Background(), BackupS3Config{Bucket: "updated", AccessKeyID: "AKID"})
	require.NoError(t, err)
	raw, err := repo.GetValue(context.Background(), settingKeyBackupS3Config)
	require.NoError(t, err)
	require.Contains(t, raw, `"secret_access_key":"ENC:secret123"`)
	cfg, err := svc.loadS3Config(context.Background())
	require.NoError(t, err)
	require.Equal(t, "secret123", cfg.SecretAccessKey)
}
