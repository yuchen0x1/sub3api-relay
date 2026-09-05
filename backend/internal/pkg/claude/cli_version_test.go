package claude

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsSupportedCLIVersion(t *testing.T) {
	valid := []string{"2.1.229", "2.1.251", "2.2.0", "3.0.1"}
	for _, v := range valid {
		require.Truef(t, IsSupportedCLIVersion(v), "version %q should be supported", v)
	}

	invalid := []string{
		"",              // 空值：未配置
		"2.1.228",       // 低于内置基线
		"2.1.229-local", // 预发布后缀会被指纹正则拒绝
		"2.1.229+build", // 构建元数据
		"v2.1.251",      // v 前缀
		"2.1",           // 省略段
		"2.1.251.1",     // 四段
		"banana",        // 非版本
	}
	for _, v := range invalid {
		require.Falsef(t, IsSupportedCLIVersion(v), "version %q should be rejected", v)
	}
}

func TestResolveCLIVersion(t *testing.T) {
	t.Run("empty falls back to builtin", func(t *testing.T) {
		require.Equal(t, CLICurrentVersion, resolveCLIVersion(""))
		require.Equal(t, CLICurrentVersion, resolveCLIVersion("   "))
	})

	t.Run("valid override wins", func(t *testing.T) {
		require.Equal(t, "2.1.251", resolveCLIVersion(" 2.1.251 "))
	})

	t.Run("invalid override falls back", func(t *testing.T) {
		require.Equal(t, CLICurrentVersion, resolveCLIVersion("9.9.9-rc1"))
		require.Equal(t, CLICurrentVersion, resolveCLIVersion("banana"))
	})
}

func TestResolvedCLIVersionMatchesCurrentEnv(t *testing.T) {
	// resolvedCLIVersion 在包初始化时解析一次；本测试进程未设置覆盖时应等于基线。
	t.Setenv(CLIVersionEnv, "")
	require.Equal(t, CLICurrentVersion, CLIVersion())
	_ = os.Getenv
}
