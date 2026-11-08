export default {

  // Subscription Progress (Header component)
  subscriptionProgress: {
    title: 'マイサブスクリプション',
    viewDetails: 'サブスクリプション詳細を見る',
    activeCount: '有効なサブスクリプション {count} 件',
    daily: '日次',
    weekly: '週次',
    monthly: '月額',
    daysRemaining: '残り {days} 日',
    expired: '期限切れ',
    expiresToday: '本日期限切れ',
    expiresTomorrow: '明日期限切れ',
    viewAll: 'すべてのサブスクリプションを見る',
    noSubscriptions: '有効なサブスクリプションはありません',
    unlimited: '無制限'
  },

  // Version Badge
  version: {
    currentVersion: '現在のバージョン',
    latestVersion: '最新バージョン',
    upToDate: '最新バージョンです',
    updateCheckWarning: '更新の確認に再試行が必要です',
    updateAvailable: '新しいバージョンがあります！',
    releaseNotes: '更新履歴',
    noReleaseNotes: '更新履歴はありません',
    viewUpdate: '更新を見る',
    viewRelease: 'リリースを見る',
    viewChangelog: '更新履歴を見る',
    openReleasePage: 'GitHub リリースページを開く',
    refresh: '更新',
    sourceMode: 'ソースビルド',
    sourceModeHint: 'ソースビルドの場合は git pull で更新してください',
    dockerModeHint: 'Docker デプロイではコンテナ内でプログラムを永続的に差し替えることはできません。Compose ディレクトリで新しいイメージを取得し、コンテナを再作成してください。',
    updateNow: '今すぐ更新',
    updating: '更新中...',
    updateComplete: '更新が完了しました',
    updateFailed: '更新に失敗しました',
    restartRequired: '更新を適用するにはサービスを再起動してください',
    restartNow: '今すぐ再起動',
    restarting: '再起動中...',
    retry: '再試行',
    rollback: 'バージョンのロールバック',
    rollbackSelectVersion: 'ロールバック先バージョンを選択（直近 3 バージョン）',
    rollbackConfirm: '{version} にロールバック',
    rollbackWarning: 'ロールバックにより選択したバージョンをダウンロードして現在のプログラムを置き換えます。完了後にサービスの再起動が必要です',
    rollingBack: 'ロールバック中...',
    rollbackComplete: 'ロールバックが完了しました',
    rollbackFailed: 'ロールバックに失敗しました',
    manualRollbackCommand: '手動ロールバック方法',
    copyCommand: 'コピー',
    copied: 'コピーしました',
    noRollbackVersions: 'ロールバックできるバージョンがありません',
    loadVersionsFailed: 'バージョン一覧の取得に失敗しました',
    rollbackSourceHint: 'ソースビルドではオンラインロールバックに対応していません',
    deployScript: 'スクリプトデプロイ',
    deployDocker: 'Docker',
    dockerEditCompose: 'docker-compose.yml のイメージバージョンを変更',
    dockerRecreate: 'コンテナを再作成'
  },

  // Recharge / Subscription Page
  purchase: {
    title: 'チャージ/サブスクリプション',
    description: '埋め込みページからチャージ/サブスクリプションを完了します',
    openInNewTab: '新しいウィンドウで開く',
    notEnabledTitle: 'この機能は無効です',
    notEnabledDesc: '管理者がチャージ/サブスクリプション入口を有効にする前まで、管理者にお問い合わせください。',
    notConfiguredTitle: 'チャージ/サブスクリプションのリンクが未設定です',
    notConfiguredDesc: '管理者が入口を有効にしましたが、リンクが未設定です。管理者にお問い合わせください。'
  },

  // Custom Page (iframe embed)
  customPage: {
    title: 'カスタムページ',
    openInNewTab: '新しいウィンドウで開く',
    notFoundTitle: 'ページが存在しません',
    notFoundDesc: 'このカスタムページは存在しないか、削除されました。',
    notConfiguredTitle: 'ページのリンクが未設定です',
    notConfiguredDesc: 'このカスタムページの URL が正しく設定されていません。',
    tableOfContents: '目次',
    copyCode: 'コピー',
    copiedCode: 'コピーしました',
    copyCodeFailed: '失敗しました'
  },

  // Announcements Page
  announcements: {
    title: 'お知らせ',
    description: 'システムのお知らせを表示します',
    unreadOnly: '未読のみ表示',
    markRead: '既読にする',
    markAllRead: 'すべて既読',
    viewAll: 'すべてのお知らせを見る',
    markedAsRead: '既読にしました',
    allMarkedAsRead: 'すべてのお知らせを既読にしました',
    newCount: '新しいお知らせが {count} 件あります',
    readAt: '既読日時',
    read: '既読',
    unread: '未読',
    startsAt: '開始日時',
    endsAt: '終了日時',
    empty: 'お知らせはありません',
    emptyUnread: '未読のお知らせはありません',
    total: '件のお知らせ',
    emptyDescription: '現在システムのお知らせはありません',
    readStatus: 'このお知らせを既読にしました',
    markReadHint: '「既読」をクリックしてこのお知らせを既読にします'
  },

  // 站内通知中心
  notifications: {
    title: '通知センター',
    description: 'サイト内通知を表示します',
    unreadCount: '未読 {count} 件',
    markAllRead: 'すべて既読にする',
    allMarkedAsRead: 'すべての通知を既読にしました',
    markedAsRead: '既読にしました',
    empty: '通知はありません',
    emptyDescription: '現在通知はありません',
    loadMore: 'さらに読み込む',
    noMore: 'これ以上通知はありません',
    read: '既読',
    unread: '未読',
    types: {
      all: 'すべて',
      system: 'システム',
      balance_low: '残高不足',
      account_quota: 'アカウント上限',
      payment_received: 'チャージ完了',
      key_rotated: 'API Key ローテーション'
    }
  },

  // User Subscriptions Page
  userSubscriptions: {
    title: 'マイサブスクリプション',
    description: 'サブスクリプションプランと利用状況を表示します',
    noActiveSubscriptions: '有効なサブスクリプションはありません',
    noActiveSubscriptionsDesc: '有効なサブスクリプションがありません。サブスクリプションについては管理者にお問い合わせください。',
    failedToLoad: 'サブスクリプションの読み込みに失敗しました',
    status: {
      active: '有効',
      expired: '期限切れ',
      revoked: '取り消し済み'
    },
    usage: '使用量',
    expires: '有効期限',
    noExpiration: '有効期限なし',
    unlimited: '無制限',
    unlimitedDesc: 'このサブスクリプションに使用量の制限はありません',
    daily: '日次',
    weekly: '週次',
    monthly: '月額',
    daysRemaining: '残り {days} 日',
    expiresOn: '{date} 期限切れ',
    resetIn: '{time} 後にリセット',
    quotaEndsIn: 'クォータは {time} 後に終了',
    windowNotActive: '初回利用待ち',
    usageOf: '{used} / {limit} 使用済み'
  },

  // Onboarding Tour
  onboarding: {
    restartTour: 'オンボーディングをもう一度見る',
    dontShowAgain: '次回から表示しない',
    dontShowAgainTitle: 'オンボーディングを完全に無効にする',
    confirmDontShow: '本当にオンボーディングを非表示にしますか？\n\n右上部のアバターメニューからいつでも再度有効にできます。',
    confirmExit: 'オンボーディングを終了しますか？右上部のメニューからいつでも再開できます。',
    interactiveHint: 'Enter キーまたはクリックで次へ',
    navigation: {
      flipPage: 'ページを切り替え',
      exit: '終了'
    },
    // Admin tour steps
    admin: {
      welcome: {
        title: '👋 Sub3API へようこそ',
        description:
          '<div style="line-height: 1.8;"><p style="margin-bottom: 16px;">Sub3API は強力な AI サービス中継プラットフォームで、AI サービスの管理と配布を簡単に行えます。</p><p style="margin-bottom: 12px;"><b>🎯 主な機能：</b></p><ul style="margin-left: 20px; margin-bottom: 16px;"><li>📦 <b>グループ管理</b> - 異なるサービスプラン（VIP、無料トライアルなど）を作成</li><li>🔗 <b>アカウントプール</b> - 複数の上流 AI サービスプロバイダーのアカウントを接続</li><li>🔑 <b>キー配布</b> - ユーザーごとに独立した API Key を生成</li><li>💰 <b>課金管理</b> - 柔軟な料率とクォータの制御</li></ul><p style="color: rgb(var(--color-success)); font-weight: 600;">続いて、3 分で初期設定を完了させましょう →</p></div>',
        nextBtn: '設定を開始 🚀',
        prevBtn: 'スキップ'
      },
      groupManage: {
        title: '📦 第一ステップ：グループ管理',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>グループとは？</b></p><p style="margin-bottom: 12px;">グループは Sub3API の中核となる概念で、次のような「サービスプラン」のことです：</p><ul style="margin-left: 20px; margin-bottom: 12px; font-size: 13px;"><li>🎯 1 つのグループに複数の上流アカウントを含められる</li><li>💰 グループごとに独立した課金倍率を持つ</li><li>👥 公開グループまたは限定グループに設定できる</li></ul><p style="margin-top: 12px; padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 例：</b>「VIP 専用回線」（高倍率）と「無料トライアル」（低倍率）の 2 つのグループを作成できます</p><p style="margin-top: 16px; color: rgb(var(--color-success)); font-weight: 600;">👉 左側の「グループ管理」をクリックして開始</p></div>'
      },
      createGroup: {
        title: '➕ 新しいグループを作成',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">最初のグループを作成しましょう。</p><p style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📝 ヒント：</b>まずテスト用グループを作成し、流れを慣れてから本番用グループを作成することをお勧めします</p><p style="color: rgb(var(--color-success)); font-weight: 600;">👉 「グループを作成」ボタンをクリック</p></div>'
      },
      groupName: {
        title: '✏️ 1. グループ名',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">グループに識別しやすい名前を付けましょう。</p><div style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>💡 名前の付け方の例：</b><ul style="margin: 8px 0 0 16px;"><li>「テストグループ」 - テスト用</li><li>「VIP 専用回線」 - 高品質なサービス</li><li>「無料トライアル」 - お試し用</li></ul></div><p style="font-size: 13px; color: var(--color-text-muted);">入力後、「次へ」をクリックして続行</p></div>',
        nextBtn: '次へ'
      },
      groupPlatform: {
        title: '🤖 2. プラットフォームを選択',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">このグループが対応する AI プラットフォームを選択します。</p><div style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📌 プラットフォームについて：</b><ul style="margin: 8px 0 0 16px;"><li><b>OpenAI</b> - GPT シリーズモデル</li><li><b>Google</b> - Gemini シリーズモデル</li></ul></div><p style="font-size: 13px; color: var(--color-text-muted);">1 つのグループで選択できるプラットフォームは 1 つだけです</p></div>',
        nextBtn: '次へ'
      },
      groupMultiplier: {
        title: '💰 3. 料率倍率',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">このグループの課金倍率を設定し、ユーザーの実際の請求額を制御します。</p><div style="padding: 8px 12px; background: rgb(var(--color-warning-soft)); border-left: 3px solid rgb(var(--color-warning)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚙️ 課金のルール：</b><ul style="margin: 8px 0 0 16px;"><li><b>1.0</b> - 原価ベースで課金</li><li><b>1.5</b> - ユーザーが $1 消費した場合、$1.5 を控除</li><li><b>2.0</b> - ユーザーが $1 消費した場合、$2 を控除</li><li><b>0.8</b> - 補助モード（赤字運営）</li></ul></div><p style="font-size: 13px; color: var(--color-text-muted);">テストグループは 1.0 の設定を推奨します</p></div>',
        nextBtn: '次へ'
      },
      groupExclusive: {
        title: '🔒 4. 限定グループ（任意）',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">グループの公開範囲とアクセス権限を制御します。</p><div style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🔐 権限について：</b><ul style="margin: 8px 0 0 16px;"><li><b>オフ</b> - 公開グループで、全ユーザーが閲覧可能</li><li><b>オン</b> - 限定グループで、指定したユーザーだけが閲覧可能</li></ul></div><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 活用例：</b>VIP ユーザー限定、内部テスト、特定顧客など</p></div>',
        nextBtn: '次へ'
      },
      groupSubmit: {
        title: '✅ グループを保存',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">内容を確認したら、作成ボタンをクリックしてグループを保存します。</p><p style="padding: 8px 12px; background: rgb(var(--color-warning-soft)); border-left: 3px solid rgb(var(--color-warning)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ 注意：</b>グループ作成後、プラットフォームの種類は変更できませんが、他の情報はいつでも編集できます</p><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>📌 次の一手：</b>作成が成功したら、上流アカウントをこのグループに追加します</p><p style="margin-top: 12px; color: rgb(var(--color-success)); font-weight: 600;">👉 「作成」ボタンをクリック</p></div>'
      },
      accountManage: {
        title: '🔗 第二ステップ：アカウントの追加',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>素晴らしい！グループの作成に成功しました 🎉</b></p><p style="margin-bottom: 12px;">続いて上流 AI サービスプロバイダーのアカウントを追加し、グループが実際にサービスを提供できるようにします。</p><div style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🔑 アカウントの役割：</b><ul style="margin: 8px 0 0 16px;"><li>上流 AI サービス（Claude、GPT など）への接続</li><li>1 つのグループに複数のアカウントを含められる（負荷分散）</li><li>OAuth と Session Key の 2 つの方式に対応</li></ul></div><p style="margin-top: 16px; color: rgb(var(--color-success)); font-weight: 600;">👉 左側の「アカウント管理」をクリック</p></div>'
      },
      createAccount: {
        title: '➕ 新しいアカウントを追加',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">ボタンをクリックして、最初の上流アカウントを追加しましょう。</p><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 ヒント：</b>より安全でキーを手動で取り出す必要もない OAuth 方式の使用をお勧めします</p><p style="margin-top: 12px; color: rgb(var(--color-success)); font-weight: 600;">👉 「アカウントを追加」ボタンをクリック</p></div>'
      },
      accountName: {
        title: '✏️ 1. アカウント名',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">アカウントに識別しやすい名前を付けましょう。</p><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 名前の付け方の例：</b>「Claude メインアカウント」「GPT 予備 1」「テストアカウント」など</p></div>',
        nextBtn: '次へ'
      },
      accountPlatform: {
        title: '🤖 2. プラットフォームを選択',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">このアカウントに対応するサービスプロバイダーのプラットフォームを選択します。</p><p style="padding: 8px 12px; background: rgb(var(--color-warning-soft)); border-left: 3px solid rgb(var(--color-warning)); border-radius: 4px; font-size: 13px;"><b>⚠️ 重要：</b>プラットフォームは、先ほど作成したグループのプラットフォームと一致している必要があります</p></div>',
        nextBtn: '次へ'
      },
      accountType: {
        title: '🔐 3. 認可方式',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">アカウントの認可方式を選択します。</p><div style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>✅ 推奨：OAuth 方式</b><ul style="margin: 8px 0 0 16px;"><li>キーを手動で取り出す必要がない</li><li>より安全で、自動更新に対応</li><li>Claude Code、ChatGPT OAuth などに適している</li></ul></div><div style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px;"><b>📌 Session Key 方式</b><ul style="margin: 8px 0 0 16px;"><li>ブラウザから手動で取り出す必要がある</li><li>定期的に更新が必要な場合がある</li><li>OAuth に対応しないプラットフォームに適している</li></ul></div></div>',
        nextBtn: '次へ'
      },
      accountPriority: {
        title: '⚖️ 4. 優先度（任意）',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">アカウントの呼び出し優先順位を設定します。</p><div style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📊 優先度のルール：</b><ul style="margin: 8px 0 0 16px;"><li>数値が小さいほど優先度が高い</li><li>システムは数値の小さいアカウントを優先して使用する</li><li>優先度が同じ場合はランダムに選択される</li></ul></div><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 活用例：</b>メインアカウントには小さな値、予備アカウントには大きな値を設定</p></div>',
        nextBtn: '次へ'
      },
      accountGroups: {
        title: '🎯 5. グループの割り当て',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>重要なステップです！</b>アカウントを先ほど作成したグループに割り当てます。</p><div style="padding: 8px 12px; background: rgb(var(--color-danger-soft)); border-left: 3px solid rgb(var(--color-danger)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ 重要なお知らせ：</b><ul style="margin: 8px 0 0 16px;"><li>少なくとも 1 つのグループを選択する必要があります</li><li>グループに割り当てられていないアカウントは使用できません</li><li>1 つのアカウントを複数のグループに割り当てられます</li></ul></div><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 ヒント：</b>先ほど作成したテストグループを選択してください</p></div>',
        nextBtn: '次へ'
      },
      accountSubmit: {
        title: '✅ アカウントを保存',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">内容を確認したら、保存ボタンをクリックします。</p><div style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📌 OAuth 認可の流れ：</b><ul style="margin: 8px 0 0 16px;"><li>保存をクリックするとサービスプロバイダーのページに移動します</li><li>サービスプロバイダーのページでログインと認可を完了します</li><li>認可が成功すると自動的に戻ります</li></ul></div><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>📌 次の一手：</b>アカウントの追加が完了したら、API キーを作成します</p><p style="margin-top: 12px; color: rgb(var(--color-success)); font-weight: 600;">👉 「保存」ボタンをクリック</p></div>'
      },
      keyManage: {
        title: '🔑 第三ステップ：キーの生成',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>おめでとうございます！アカウントの設定が完了しました 🎉</b></p><p style="margin-bottom: 12px;">最後のステップです。API Key を生成して、サービスが正常に動作するか確認します。</p><div style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🔑 API Key の役割：</b><ul style="margin: 8px 0 0 16px;"><li>AI サービスを呼び出すための認証情報</li><li>各キーは 1 つのグループに紐づく</li><li>クォータと有効期限を設定できる</li><li>個別の利用統計に対応</li></ul></div><p style="margin-top: 16px; color: rgb(var(--color-success)); font-weight: 600;">👉 左側の「API キー」をクリック</p></div>'
      },
      createKey: {
        title: '➕ キーの作成',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">ボタンをクリックして、最初の API Key を作成します。</p><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 ヒント：</b>作成後はすぐにコピーして保存してください。キーは一度しか表示されません</p><p style="margin-top: 12px; color: rgb(var(--color-success)); font-weight: 600;">👉 「キーを作成」ボタンをクリック</p></div>'
      },
      keyName: {
        title: '✏️ 1. キー名',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">キーに管理しやすい名前を付けましょう。</p><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 名前の付け方の例：</b>「テストキー」「本番環境」「モバイルアプリ」など</p></div>',
        nextBtn: '次へ'
      },
      keyGroup: {
        title: '🎯 2. グループを選択',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">先ほど設定したグループを選択します。</p><div style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📌 グループで決まること：</b><ul style="margin: 8px 0 0 16px;"><li>このキーで使用できるアカウント</li><li>課金の倍率</li><li>限定キーかどうか</li></ul></div><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 ヒント：</b>先ほど作成したテストグループを選択してください</p></div>',
        nextBtn: '次へ'
      },
      keySubmit: {
        title: '🎉 生成してコピー',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">作成をクリックすると、完全な API Key が生成されます。</p><div style="padding: 8px 12px; background: rgb(var(--color-danger-soft)); border-left: 3px solid rgb(var(--color-danger)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ 重要なお知らせ：</b><ul style="margin: 8px 0 0 16px;"><li>キーは一度しか表示されません。すぐにコピーしてください</li><li>失った場合は再生成が必要です</li><li>厳重に保管し、他者に漏らさないでください</li></ul></div><div style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🚀 次の一手：</b><ul style="margin: 8px 0 0 16px;"><li>生成された sk-xxx のキーをコピーする</li><li>OpenAI インターフェースに対応したクライアントで使用する</li><li>AI サービスの利用を始める！</li></ul></div><p style="margin-top: 12px; color: rgb(var(--color-success)); font-weight: 600;">👉 「作成」ボタンをクリック</p></div>'
      }
    },
    // User tour steps
    user: {
      welcome: {
        title: '👋 Sub3API へようこそ',
        description:
          '<div style="line-height: 1.8;"><p style="margin-bottom: 16px;">こんにちは！Sub3API AI サービスプラットフォームへようこそ。</p><p style="margin-bottom: 12px;"><b>🎯 クイックスタート：</b></p><ul style="margin-left: 20px; margin-bottom: 16px;"><li>🔑 API キーを作成する</li><li>📋 キーをアプリにコピーする</li><li>🚀 AI サービスの利用を開始する</li></ul><p style="color: rgb(var(--color-success)); font-weight: 600;">1 分で始めましょう →</p></div>',
        nextBtn: '開始 🚀',
        prevBtn: 'スキップ'
      },
      keyManage: {
        title: '🔑 API キー管理',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">ここですべての API アクセスキーを管理できます。</p><p style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px;"><b>📌 API キーとは？</b><br/>API キーは AI サービスへのアクセス権である認証情報で、アプリの AI 機能を利用できるようにする鍵のようなものです。</p><p style="margin-top: 12px; color: rgb(var(--color-success)); font-weight: 600;">👉 クリックしてキーページへ</p></div>'
      },
      createKey: {
        title: '➕ 新しいキーの作成',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">ボタンをクリックして、最初の API キーを作成します。</p><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 ヒント：</b>キーは作成後に一度しか表示されません。必ずコピーして保存してください</p><p style="margin-top: 12px; color: rgb(var(--color-success)); font-weight: 600;">👉 「キーを作成」をクリック</p></div>'
      },
      keyName: {
        title: '✏️ キー名',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">キーに識別しやすい名前を付けましょう。</p><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>💡 例：</b>「私の最初のキー」「テスト用」など</p></div>',
        nextBtn: '次へ'
      },
      keyGroup: {
        title: '🎯 グループを選択',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">管理者が割り当てたサービスグループを選択します。</p><p style="padding: 8px 12px; background: rgb(var(--color-info-soft)); border-left: 3px solid rgb(var(--color-info)); border-radius: 4px; font-size: 13px;"><b>📌 グループについて：</b><br/>グループによってサービス品質や課金の基準が異なる場合があるため、必要に応じて選択してください。</p></div>',
        nextBtn: '次へ'
      },
      keySubmit: {
        title: '🎉 作成完了',
        description:
          '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">クリックして API キーの作成を確定します。</p><div style="padding: 8px 12px; background: rgb(var(--color-danger-soft)); border-left: 3px solid rgb(var(--color-danger)); border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ 重要：</b><ul style="margin: 8px 0 0 16px;"><li>作成後すぐにキー（sk-xxx）をコピーしてください</li><li>キーは一度しか表示されず、失った場合は再生成が必要です</li></ul></div><p style="padding: 8px 12px; background: rgb(var(--color-success-soft)); border-left: 3px solid rgb(var(--color-success)); border-radius: 4px; font-size: 13px;"><b>🚀 使用方法：</b><br/>OpenAI インターフェースに対応した任意のクライアント（ChatBox、OpenCat など）にキーを設定すれば、利用を開始できます！</p><p style="margin-top: 12px; color: rgb(var(--color-success)); font-weight: 600;">👉 「作成」ボタンをクリック</p></div>'
      }
    }
  },

  // Payment System
  payment: {
    title: 'チャージ/サブスクリプション',
    amountLabel: 'チャージ金額',
    paymentAmount: '支払金額',
    creditedBalance: '入金額',
    quickAmounts: 'クイック金額',
    customAmount: '任意の金額',
    enterAmount: '金額を入力',
    paymentMethod: '支払い方法',
    fee: '手数料',
    actualPay: '実際のお支払い金額',
    createOrder: '支払を確定',
    methods: {
      easypay: 'EasyPay',
      alipay: 'Alipay',
      wxpay: 'WeChat Pay',
      stripe: 'Stripe',
      airwallex: 'Airwallex',
      card: 'クレジットカード',
      link: 'Link',
      alipay_direct: 'Alipay（直接接続）',
      wxpay_direct: 'WeChat Pay（直接接続）',
    },
    status: {
      pending: '支払い待ち',
      paid: '支払い済み',
      recharging: 'チャージ処理中',
      completed: '完了',
      expired: '期限切れ',
      cancelled: 'キャンセル済み',
      failed: '失敗',
      refund_requested: '返金申請中',
      refunding: '返金処理中',
      refund_pending: '返金処理待ち',
      refunded: '返金済み',
      partially_refunded: '一部返金済み',
      refund_failed: '返金失敗',
    },
    qr: {
      scanToPay: 'QRコードを読み取って支払ってください',
      scanAlipay: 'Alipay のQRコードで支払',
      scanWxpay: 'WeChat のQRコードで支払',
      scanAlipayHint: 'スマホで Alipay を起動し、QRコードを読み取ってください',
      scanWxpayHint: 'スマホで WeChat を起動し、QRコードを読み取ってください',
      payInNewWindow: '新しいウィンドウで支払を完了してください',
      payInNewWindowHint: '支払いページが新しいウィンドウで開きました。支払を完了してからこのページに戻ってください',
      openPayWindow: '支払いページを開き直す',
      expiresIn: '支払い残り時間',
      expired: '注文が期限切れです',
      expiredDesc: '注文がタイムアウトしました。新しい注文を作成してください',
      cancelled: '注文がキャンセルされました',
      cancelledDesc: 'お支払いをキャンセルしました',
      waitingPayment: 'お支払いを待っています...',
      cancelOrder: '注文をキャンセル',
      alipayOpening: 'Alipay を開いています',
      alipayContinueInApp: 'Alipay 内で支払を完了してください',
      alipayWaitingHint: '支払い結果はサーバーで確認され、このページは自動的に更新されます',
      alipayFallbackTitle: 'Alipay を開けませんでした',
      alipayFallbackHint: 'Alipay を開き直すか、下の QR コードを保存して Alipay のアルバムから認識してください',
      reopenAlipay: 'Alipay を開き直す',
      saveQRCode: 'QR コードを保存',
      alipaySaveAndScanHint: 'QR コードを保存し、Alipay の「スキャン」を開いてアルバムから QR コードを選択してください',
    },
    orders: {
      title: 'マイオーダー',
      empty: '注文はありません',
      orderId: '注文 ID',
      orderNo: '注文番号',
      amount: '金額',
      payAmount: '実際の支払い',
      creditedAmount: '入金額',
      fee: '手数料',
      baseAmount: 'チャージ金額',
      includedInPayAmount: '実際のお支払い金額に含まれます',
      status: 'ステータス',
      paymentMethod: '支払い方法',
      createdAt: '作成日時',
      cancel: '注文をキャンセル',
      userId: 'ユーザー ID',
      orderType: '注文タイプ',
      actions: '操作',
      requestRefund: '返金を申請',
    },
    result: {
      success: '支払いが完了しました',
      subscriptionSuccess: 'サブスクリプションが完了しました',
      processing: '支払い処理中',
      processingHint: '支払い結果は確認中です。ページは自動的に更新されます。',
      failed: '支払いに失敗しました',
      backToRecharge: 'チャージに戻る',
      viewOrders: '注文を見る',
    },
    currentBalance: '現在の残高',
    groupFallback: 'グループ #{id}',
    rechargeAccount: 'チャージアカウント',
    activeSubscription: '現在のサブスクリプション',
    noActiveSubscription: '有効なサブスクリプションはありません',
    tabTopUp: 'チャージ',
    tabSubscribe: 'サブスクリプション',
    noPlans: '利用可能なサブスクリプションプランはありません',
    notAvailable: 'チャージ機能はまだ利用できません',
    confirmSubscription: 'サブスクリプションを確定',
    confirmCancel: 'この注文をキャンセルしてもよろしいですか？',
    amountTooLow: '最低金額は {min} です',
    amountTooHigh: '最高金額は {max} です',
    amountNoMethod: 'この金額では利用可能な支払い方法がありません',
    rechargeRatePreview: '現在の倍率：1 CNY = {usd} USD',
    refundReason: '返金理由',
    refundReasonPlaceholder: '返金理由を入力してください',
    stripeLoadFailed: '決済コンポーネントの読み込みに失敗しました。ページを再読み込みして再試行してください',
    stripeMissingParams: '注文 ID または決済キーがありません',
    stripeNotConfigured: 'Stripe が設定されていません',
    airwallexLoadFailed: 'Airwallex 決済コンポーネントの読み込みに失敗しました。ページを再読み込みして再試行してください',
    airwallexMissingParams: 'Airwallex の決済パラメータがありません',
    errors: {
      tooManyPending: '支払い待ちの注文が多すぎます（最大 {max} 件）。既存の注文を完了またはキャンセルしてください',
      cancelRateLimited: '注文のキャンセルが頻繁すぎます。しばらくしてから再試行してください',
      wechatH5NotAuthorized: '現在の事業者は WeChat H5 決済を利用できないため、WeChat 内で現在のページを開いて決済を続けてください。',
      wechatPaymentMpNotConfigured: '現在のサイトは WeChat 公式アカウント / JSAPI 決済の設定が完了していないため、現時点では WeChat 内で直接決済を開始できません。',
      wechatJsapiUnavailable: '現在の環境では WeChat 決済を起動できませんでした。WeChat 内でこのページを開いていることを確認して再試行してください。',
      wechatJsapiFailed: 'WeChat 決済が完了しませんでした。決済を再度起動するか、QR コード決済をお使いください。',
      wechatUnavailable: '現在 WeChat 決済は利用できません。しばらくしてから再試行してください。',
      wechatOpenInWeChatHint: '現在のページのリンクをコピーして WeChat 内で開くか、PC 版 WeChat の QR コード決済をお使いください。',
      wechatScanOnDesktopHint: 'PC では WeChat の「scan」で QR コードを読み取って決済してください。モバイルでは WeChat 内で現在のページを開いてください。',
      wechatSwitchBrowserHint: 'PC 版 WeChat の QR コード決済に変更するか、外部ブラウザでこのページを開き直して再試行してください。',
      mobilePaymentFallbackToQr: '現在の事業者はモバイル決済を利用できないため、QR コード決済に自動的に切り替えました。',
      alipayDesktopUnavailable: '現在の Alipay デスクトップ決済では QR コードを生成できませんでした。',
      alipayDesktopQrHint: 'PC 版の Alipay では QR コード決済が表示されます。再読み込みして再試行するか、ブラウザが決済ページをブロックしていないか確認してください。',
      alipayMobileUnavailable: '現在のページから Alipay への遷移に成功しませんでした。',
      alipayMobileOpenHint: '現在のページで Alipay アプリを開くことを許可するか、システムブラウザから再度決済を開始してください。',
      // Structured error codes (reason strings from backend ApplicationError)
      PAYMENT_DISABLED: '決済システムは無効です',
      USER_INACTIVE: 'アカウントは無効化されています',
      BALANCE_PAYMENT_DISABLED: '残高チャージ機能は無効です',
      INVALID_AMOUNT: '金額が不正です',
      INVALID_INPUT: 'パラメータに誤りがあります',
      PLAN_NOT_AVAILABLE: 'プランが存在しないか、公開終了しています',
      GROUP_NOT_FOUND: 'サブスクリプショングループが利用できません',
      GROUP_TYPE_MISMATCH: 'グループの種類がサブスクリプションタイプではありません',
      TOO_MANY_PENDING: '支払い待ちの注文が多すぎます（最大 {max} 件）。既存の注文を完了またはキャンセルしてください',
      DAILY_LIMIT_EXCEEDED: '本日のチャージ上限に達しました。残り {remaining}',
      PAYMENT_GATEWAY_ERROR: '支払い方法を利用できません',
      NO_AVAILABLE_INSTANCE: '利用可能な決済チャネルがありません',
      PAYMENT_PROVIDER_MISCONFIGURED: '決済チャネルの設定に誤りがあります。管理者にお問い合わせください',
      WXPAY_CONFIG_MISSING_KEY: 'WeChat 決済の設定に必須項目がありません：{key}',
      WXPAY_CONFIG_INVALID_KEY_LENGTH: 'WeChat 決済の {key} の長さが不正です。{expected} バイトである必要があります（実際 {actual}）',
      WXPAY_CONFIG_INVALID_KEY: 'WeChat 決済の {key} の形式が不正です。PEM 内容が完全にコピーされているか確認してください',
      PENDING_ORDERS: 'この事業者には未完了の注文があります。注文が完了してから操作してください',
      PAYMENT_PROVIDER_CONFLICT: 'この支払い方法には他の有効な事業者インスタンスが既にあります。先に無効化してから続行してください。',
      CANCEL_RATE_LIMITED: '注文のキャンセルが頻繁すぎます。しばらくしてから再試行してください',
      NOT_FOUND: '注文が存在しません',
      FORBIDDEN: 'この注文を操作する権限がありません',
      CONFLICT: '注文のステータスが変更されました。再読み込みしてください',
      INVALID_ORDER_TYPE: '残高注文のみ返金を申請できます',
      INVALID_STATUS: '現在の注文ステータスではこの操作は実行できません',
      BALANCE_NOT_ENOUGH: '返金額が残高を超えています',
      REFUND_AMOUNT_EXCEEDED: '返金額がチャージ金額を超えています',
      REFUND_FAILED: '返金に失敗しました',
    },
    airwallexPay: 'Airwallex で支払う',
    stripePay: '今すぐ支払う',
    stripeSuccessProcessing: '支払いが完了しました。注文を処理しています...',
    stripePopup: {
      redirecting: '支払いページに移動しています...',
      loadingQr: 'WeChat 決済の QR コードを取得しています...',
      timeout: '決済認証情報の待機にタイムアウトしました。再試行してください',
      qrFailed: 'WeChat 決済の QR コードを取得できませんでした',
    },
    subscribeNow: '今すぐ開通',
    renewNow: '更新',
    selectPlan: 'プランを選択',
    planFeatures: '機能',
    planCard: {
      rate: '倍率',
      peakRate: 'ピーク倍率',
      dailyLimit: '日次上限',
      weeklyLimit: '週次上限',
      monthlyLimit: '月次上限',
      quota: 'クォータ',
      unlimited: '無制限',
      models: 'モデル',
    },
    days: '日',
    weeks: '週間',
    months: 'か月',
    years: '年',
    oneMonth: '1 か月',
    oneYear: '1 年',
    perMonth: '月',
    perYear: '年',
    admin: {
      tabs: {
        overview: '概要',
        orders: '注文管理',
        channels: '決済チャネル',
        plans: 'サブスクリプションプラン',
      },
      todayRevenue: '本日の売上',
      totalRevenue: '総売上',
      todayOrders: '本日の注文',
      orderCount: '注文数',
      avgAmount: '平均金額',
      revenue: '売上',
      dailyRevenue: '日次売上',
      paymentDistribution: '支払い方法の分布',
      colUser: 'ユーザー',
      topUsers: '消費ランキング',
      noData: 'データがありません',
      days: '日',
      weeks: '週間',
      months: '月',
      searchOrders: '注文を検索...',
      allStatuses: 'すべてのステータス',
      allPaymentTypes: 'すべての支払い方法',
      allOrderTypes: 'すべての注文タイプ',
      orderDetail: '注文詳細',
      orderType: '注文タイプ',
      orders: '注文',
      balanceOrder: '残高チャージ',
      subscriptionOrder: 'サブスクリプション',
      paidAt: '支払い日時',
      completedAt: '完了日時',
      expiresAt: '有効期限',
      feeRate: '手数料率',
      refund: '返金',
      refundOrder: '返金注文',
      refundAmount: '返金金額',
      maxRefundable: '返金可能上限額',
      refundReason: '返金理由',
      refundReasonPlaceholder: '返金理由を入力してください',
      confirmRefund: '返金を確定',
      refundSuccess: '返金が完了しました',
      refundPending: '返金処理中、ゲートウェイの確認待ち',
      queryRefundStatus: '返金ステータスを確認',
      refundInfo: '返金情報',
      refundEnabled: '返金を許可',
      allowUserRefund: 'ユーザーによる返金を許可',
      alreadyRefunded: '返金済み',
      deductBalance: '残高から控除',
      deductBalanceHint: 'ユーザーの残高からチャージ金額を差し引きます',
      userBalance: 'ユーザー残高',
      orderAmount: '注文金額',
      insufficientBalance: '残高が不足するため、$0 まで控除されます',
      noDeduction: 'ユーザー残高は控除されません',
      forceRefund: '強制返金（残高チェックを無視）',
      orderCancelled: '注文はキャンセルされました',
      retry: '再試行',
      retrySuccess: '再試行に成功しました',
      approveRefund: '返金を承認',
      retryRefund: '返金を再試行',
      refundRequestInfo: '返金申請情報',
      refundRequestedAt: '申請日時',
      refundRequestedBy: '申請者',
      refundRequestReason: '申請理由',
      auditLogs: '操作ログ',
      operator: '操作者',
      channelName: 'チャネル名',
      channelDescription: 'チャネル説明',
      createChannel: 'チャネルを作成',
      editChannel: 'チャネルを編集',
      deleteChannel: 'チャネルを削除',
      deleteChannelConfirm: 'このチャネルを削除してもよろしいですか？',
      planName: 'プラン名',
      planDescription: 'プラン説明',
      createPlan: 'プランを作成',
      editPlan: 'プランを編集',
      deletePlan: 'プランを削除',
      deletePlanConfirm: 'このプランを削除してもよろしいですか？',
      originalPrice: '原価',
      price: '価格',
      currency: '通貨表記',
      currencyPlaceholder: '例：USD / NZD / CNY',
      currencyHint: '価格表示のみに使用する ISO 3 文字通貨コードです。空欄の場合は表示されません。実際の請求には影響しません',
      subscriptionCnyPayPreview: 'CNY チャネルの実際請求額プレビュー：{amount}',
      subscriptionCnyPayPreviewWithFee: '（手数料 {feeRate}% を含む：{total}）',
      validity: '有効期間',
      validityUnit: '有効期間の単位',
      sortOrder: '並び順',
      forSale: '公開ステータス',
      onSale: '公開中',
      offSale: '公開終了',
      group: 'グループ',
      groupId: 'グループ ID',
      features: '機能',
      featuresHint: '1 行につき 1 機能',
      featuresPlaceholder: 'プランの機能を入力...',
      providerManagement: '事業者管理',
      providerManagementDesc: '決済事業者のインスタンスを管理します',
      createProvider: '事業者を作成',
      editProvider: '事業を編集',
      deleteProvider: '事業者を削除',
      deleteProviderConfirm: 'この事業者を削除してもよろしいですか？',
      providerName: '事業者名',
      providerKey: '事業者識別子',
      selectProviderKey: '事業者識別子を選択',
      providerConfig: '事業者設定',
      noProviders: '事業者がありません',
      noProvidersHint: '決済の受付を開始するには、事業者インスタンスを作成してください',
      supportedTypes: '対応する支払い方法',
      supportedTypesHint: 'この事業者が対応する支払い方法を選択します',
      rateMultiplier: '料率倍率',
      dashboardTitle: '決済概要',
      dashboardDesc: 'チャージ注文の統計と分析',
      daySuffix: '日',
      paymentConfigTitle: '決済設定',
      paymentConfigDesc: '決済事業者と関連設定を管理します',
      plansPageTitle: 'サブスクリプションプラン管理',
      plansPageDesc: 'サブスクリプションプランの設定を管理します',
      tabPlanConfig: 'プラン設定',
      tabUserSubs: 'ユーザーサブスクリプション',
      selectGroup: 'グループを選択してください',
      groupRequired: 'サブスクリプショングループを選択してください',
      priceRequired: '価格は 0 より大きい必要があります',
      validityRequired: '有効期間は 0 より大きい必要があります',
      groupMissing: '未設定',
      groupInfo: 'グループ情報',
      platform: 'プラットフォーム',
      rateMultiplierLabel: '倍率',
      dailyLimit: '日次上限',
      weeklyLimit: '週次上限',
      monthlyLimit: '月次上限',
      unlimited: '無制限',
      searchUserSubs: 'ユーザーサブスクリプションを検索...',
      daily: '日',
      weekly: '週間',
      monthly: '月',
      subsStatus: {
        active: '有効',
        expired: '期限切れ',
        revoked: '取り消し済み',
      },
    },
  },

}
