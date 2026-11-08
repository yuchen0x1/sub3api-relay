export default {
  audit: {
    title: '操作ログ',
    description: 'サイト運営者とユーザーの管理画面操作を記録します。リクエストヘッダーの認証情報は前後のみ保持し、リクエストボディはマスク処理済みです。ログは1件ずつ削除できません。全量削除はセキュリティ設定の管理下にあります。',
    clearAll: '全量クリア',
    empty: '操作ログはありません',
    loadFailed: '操作ログの読み込みに失敗しました',
    filters: {
      all: 'すべて',
      q: 'キーワード',
      qPlaceholder: 'パス / アクション / 操作者メール',
      actorEmail: '操作者メール',
      action: 'アクション',
      clientIp: 'クライアント IP',
      method: 'リクエストメソッド',
      authMethod: '認証方式',
      result: '結果',
      resultSuccess: '成功',
      resultFailure: '失敗',
      startTime: '開始時間',
      endTime: '終了時間'
    },
    columns: {
      time: '時間',
      actor: '操作者',
      action: 'アクション',
      method: 'メソッド',
      result: '結果',
      clientIp: 'クライアント IP',
      detail: '詳細'
    },
    detail: {
      title: '操作ログ詳細',
      actorRole: 'ロール',
      methodPath: 'メソッド / パス',
      latency: '所要時間',
      requestId: 'リクエスト ID',
      credential: '認証情報（マスク）',
      userAgent: 'User-Agent',
      requestBody: 'リクエストボディ（マスク済み）',
      extra: '追加情報'
    },
    clearConfirm: {
      title: 'すべての操作ログをクリア',
      message: 'この操作はすべての操作ログを完全に削除し、元に戻すことはできません。クリア操作自体は記録として残ります。続行してもよろしいですか？',
      totpTitle: '二段階認証コードを入力',
      totpHint: '操作ログのクリアには現場での TOTP 認証コードの入力が必要です。',
      success: '{count} 件の操作ログをクリアしました',
      failed: '操作ログのクリアに失敗しました'
    }
  }
}