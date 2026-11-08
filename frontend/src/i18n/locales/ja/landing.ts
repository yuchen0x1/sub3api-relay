export default {
  batchImageGuide: {
    title: '画像一括生成',
    description: '複数のプロンプトを一度に送信し、タスク完了後に画像結果をまとめてダウンロードできます'
  },
  // Home Page
  home: {
    viewOnGithub: 'GitHub で表示',
    viewDocs: 'ドキュメントを表示',
    docs: 'ドキュメント',
    switchToLight: 'ライトモードに切り替え',
    switchToDark: 'ダークモードに切り替え',
    dashboard: 'コンソール',
    login: 'ログイン',
    getStarted: '今すぐ始める',
    goToDashboard: 'コンソールへ',
    gateway: {
      title: '1つのAPIでモデルに直結',
      description: '統一ゲートウェイ API 契約。1つのキーですべての接続済みモデルを呼び出せます',
      request: 'リクエスト例',
      response: 'レスポンス例',
      chat: 'チャット補完',
      chatDesc: 'POST /v1/messages、ストリーミング出力を標準サポート',
      models: 'モデル一覧',
      modelsDesc: 'GET /v1/models、いつでも現在利用可能なモデルカタログを取得',
      streaming: 'ストリーミング出力',
      streamingDesc: 'SSE ストリームレスポンス、上流障害時は自動フェイルオーバー',
      auth: 'キー認証',
      authDesc: 'Bearer API Key、1行のコードで既存プロジェクトに統合'
    },
    // 新增：面向用户的价值主张
    heroSubtitle: '1つのキーで、複数の AI モデルを利用',
    heroDescription: '複数のサブスクリプションアカウントを管理する必要なく、Claude、GPT、Gemini などの主要 AI サービスにワンストップでアクセス',
    tags: {
      subscriptionToApi: 'サブスクリプションを API に変換',
      stickySession: 'セッション維持',
      realtimeBilling: '従量課金'
    },
    // 用户痛点区块
    painPoints: {
      title: 'こんなお悩みはありませんか？',
      items: {
        expensive: {
          title: 'サブスクリプション費用が高い',
          desc: '各 AI サービスを個別に契約するため、月々の出費が増えていく'
        },
        complex: {
          title: '複数アカウントの管理が大変',
          desc: '異なるプラットフォームのアカウントやキーが散在し、管理が煩雑'
        },
        unstable: {
          title: 'サービスが不安定',
          desc: '単一アカウントは制限にかかりやすく、通常利用に影響する'
        },
        noControl: {
          title: '利用量を制御できない',
          desc: 'どこでお金を使ったか分からず、チームメンバーの利用も制限できない'
        }
      }
    },
    // 解决方案区块
    solutions: {
      title: '私たちが解決します',
      subtitle: 'シンプル3ステップで、AI を安心して使い始められます'
    },
    features: {
      unifiedGateway: 'ワンクリック接続',
      unifiedGatewayDesc: '1つの API キーを取得すれば、接続済みのすべての AI モデルを利用できます。個別に申請する必要はありません。',
      multiAccount: '安定・信頼性',
      multiAccountDesc: '複数の上流アカウントをインテリジェントにスケジューリングし、自動フェイルオーバーと負荷分散で、頻繁なエラーを解消します。',
      balanceQuota: 'キーと利用量の透明性',
      balanceQuotaDesc: 'キーごとにクォータと倍率を割り当て、個人、家族、小規模チームでも毎回の呼び出しコストを明確に把握できます。'
    },
    // 优势对比
    comparison: {
      title: 'なぜ私たちを選ぶのか？',
      headers: {
        feature: '比較項目',
        official: '公式サブスクリプション',
        us: '本プラットフォーム'
      },
      items: {
        pricing: {
          feature: '支払い方法',
          official: '固定月額、使わなくても支払う',
          us: '従量課金、使った分だけ支払う'
        },
        models: {
          feature: 'モデル選択',
          official: '単一プロバイダー',
          us: '複数モデルを自由に切り替え'
        },
        management: {
          feature: 'アカウント管理',
          official: '各サービスを個別に管理',
          us: '統合キーでワンストップ管理'
        },
        stability: {
          feature: 'サービス安定性',
          official: '単一アカウントは制限されやすい',
          us: 'マルチアカウントプールで自動切り替え'
        },
        control: {
          feature: '利用量の制御',
          official: '制限できない',
          us: 'クォータ設定・明細照会が可能'
        }
      }
    },
    providers: {
      title: '対応している AI モデル',
      description: '1つの API、複数の選択肢',
      supported: '対応済み',
      soon: '近日公開',
      claude: 'Claude',
      gemini: 'Gemini',
      antigravity: 'Antigravity',
      more: 'もっと見る'
    },
    // CTA 区块
    cta: {
      title: '準備はできましたか？',
      description: '登録すると無料トライアル枠を獲得でき、ワンストップの AI サービスを体験できます',
      button: '無料登録'
    },
    footer: {
      allRightsReserved: 'All rights reserved.'
    }
  },

  // Key Usage Query Page
  keyUsage: {
    title: 'API Key 利用量照会',
    subtitle: 'API Key を入力すると、リアルタイムの消費金額と利用状態を確認できます',
    placeholder: 'sk-ant-mirror-xxxxxxxxxxxx',
    query: '照会',
    querying: '照会中...',
    privacyNote: 'キーはブラウザ内でのみ処理され、保存されることはありません',
    dateRange: '集計範囲:',
    dateRangeToday: '今日',
    dateRange7d: '7 日',
    dateRange30d: '30 日',
    dateRange90d: '90 日',
    dateRangeCustom: 'カスタム',
    apply: '適用',
    used: '使用済み',
    detailInfo: '詳細情報',
    tokenStats: 'Token 統計',
    dailyDetail: '日別明細',
    modelStats: 'モデル利用量統計',
    // Table headers
    date: '日付',
    model: 'モデル',
    requests: 'リクエスト数',
    inputTokens: '入力 Tokens',
    outputTokens: '出力 Tokens',
    cacheCreationTokens: 'キャッシュ作成',
    cacheReadTokens: 'キャッシュ読み取り',
    cacheWriteTokens: 'キャッシュ書き込み',
    totalTokens: '合計 Tokens',
    cost: '費用',
    // Status
    quotaMode: 'Key クォータモード',
    walletBalance: 'ウォレット残高',
    // Ring card titles
    totalQuota: '総クォータ',
    limit5h: '5 時間制限',
    limitDaily: '日次制限',
    limit7d: '7 日制限',
    limitWeekly: '週次制限',
    limitMonthly: '月次制限',
    // Detail rows
    remainingQuota: '残クォータ',
    expiresAt: '有効期限',
    todayExpires: '(今日期限)',
    daysLeft: '({days} 日)',
    usedQuota: '使用済みクォータ',
    resetNow: 'まもなくリセット',
    subscriptionType: 'サブスクリプションタイプ',
    subscriptionExpires: 'サブスクリプション期限',
    // Usage stat cells
    todayRequests: '今日のリクエスト',
    todayInputTokens: '今日の入力',
    todayOutputTokens: '今日の出力',
    todayTokens: '今日の Tokens',
    todayCacheCreation: '今日のキャッシュ作成',
    todayCacheRead: '今日のキャッシュ読み取り',
    todayCost: '今日の費用',
    rpmTpm: 'RPM / TPM',
    totalRequests: '累計リクエスト',
    totalInputTokens: '累計入力',
    totalOutputTokens: '累計出力',
    totalTokensLabel: '累計 Tokens',
    totalCacheCreation: '累計キャッシュ作成',
    totalCacheRead: '累計キャッシュ読み取り',
    totalCost: '累計費用',
    avgDuration: '平均所要時間',
    // Messages
    enterApiKeyField: 'API Key を入力してください',
    querySuccess: '照会成功',
    queryFailed: '照会失敗',
    queryFailedRetry: '照会に失敗しました。しばらくして再試行してください',
    noDailyUsage: '日別の利用量データはありません',
  },

  // Setup Wizard
  setup: {
    title: 'Sub3API インストールウィザード',
    description: 'Sub3API インスタンスを設定します',
    database: {
      title: 'データベース設定',
      description: 'PostgreSQL データベースに接続します',
      host: 'ホスト',
      port: 'ポート',
      username: 'ユーザー名',
      passwordField: 'パスワード',
      databaseName: 'データベース名',
      sslMode: 'SSL モード',
      passwordPlaceholder: 'パスワード',
      ssl: {
        disable: '無効',
        require: '要求',
        verifyCa: 'CA を検証',
        verifyFull: '完全検証'
      }
    },
    redis: {
      title: 'Redis 設定',
      description: 'Redis サーバーに接続します',
      host: 'ホスト',
      port: 'ポート',
      username: 'ユーザー名（任意）',
      passwordField: 'パスワード（任意）',
      database: 'データベース',
      usernamePlaceholder: 'デフォルトユーザーは空欄',
      passwordPlaceholder: 'パスワード',
      enableTls: 'TLS を有効化',
      enableTlsHint: 'Redis 接続に TLS を使用します（公開 CA 証明書）'
    },
    admin: {
      title: 'サイト運営者アカウント',
      description: '一意のサイト運営者のログインアカウントとパスワードを設定します。以降はこのアカウントのみでバックエンドにアクセスします',
      ownerSetupHint: 'サーバー環境は準備完了です。ここではサイト運営者のアカウントとパスワードのみ設定します。データベースや Redis のパスワードは不要です。',
      remoteTokenHint: 'リモート初期化には、デプロイディレクトリの .env に保存されているワンタイムパスワードが必要です。',
      login: 'サイト運営者のログインアカウント',
      email: 'サイト運営者アカウント',
      passwordField: 'パスワード',
      confirmPasswordField: 'パスワードの確認',
      passwordPlaceholder: '8 文字以上',
      confirmPasswordPlaceholder: 'パスワードの確認',
      passwordMismatch: 'パスワードが一致しません'
    },
    ready: {
      title: 'インストール準備',
      description: '設定を確認してインストールを完了します',
      database: 'データベース',
      redis: 'Redis',
      adminEmail: 'サイト運営者メール'
    },
    status: {
      testing: 'テスト中...',
      success: '接続成功',
      testConnection: '接続テスト',
      installing: 'インストール中...',
      completeInstallation: 'インストール完了',
      completed: 'インストール完了！',
      redirecting: 'ログインページにリダイレクト中...',
      restarting: 'サービスを再起動しています。しばらくお待ちください...',
      timeout: 'サービスの再起動が予想より長くかかっています。ページを手動で更新してください。'
    }
  },

  // Common
}