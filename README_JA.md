<div align="center">

# Sub3API

[![Go](https://img.shields.io/badge/Go-1.26.10-00ADD8.svg)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)

<a href="https://trendshift.io/repositories/21823" target="_blank"><img src="https://trendshift.io/api/badge/repositories/21823" alt="yuchen0x1%2Fsub3api-relay | Trendshift" width="250" height="55"/></a>

**サブスクリプションクォータ配分のための AI API ゲートウェイプラットフォーム**

[English](README.md) | [中文](README_CN.md) | 日本語 | [v0.4.1 更新内容](docs/release-notes/RELEASE_NOTES_v0.4.1.md)

</div>

## v0.4.1 の主な更新

- GitHub API がレート制限された場合、更新確認は公開 Release ページへ自動的にフォールバックし、リリース資産のダウンロードも維持します。
- Docker のローカル開発ではホストのループバック公開経由で setup ウィザードを利用でき、公開バインドの初期化保護は維持されます。
- Vertex 管理 GCS bucket を設定していないローカル環境で、バッチ画像設定による再起動ループを防ぎます。
- プロキシ処理、スケジューラキャッシュ、デスクトップの接続先設定、管理画面設定の信頼性を改善しました。

[完全な v0.4.1 更新内容](docs/release-notes/RELEASE_NOTES_v0.4.1.md)。

[v0.4.1 全変更](docs/release-notes/RELEASE_NOTES_v0.4.1.md)・[ローカル準備と GitHub リリース手順](docs/RELEASING.md)・[AI 自動デプロイ](deploy/AI_DEPLOY.md)・[デプロイ手順](deploy/README.md)

## ⚠️ 重要なお知らせ

本プロジェクトをご利用になる前に、以下の内容を必ずよくお読みください：

- **🚨 利用規約のリスク**：本プロジェクトの使用は、上流プロバイダーの利用規約に違反する可能性があります。ご利用前に各プロバイダーのユーザー規約を必ずご確認ください。使用により生じるすべてのリスクはユーザーご自身が負うものとします。
- **⚖️ 法令遵守**：お住まいの国または地域の法令を遵守した上で本プロジェクトをご利用ください。いかなる違法な目的での使用も固く禁じます。
- **📖 免責事項**：本プロジェクトは技術的な学習および研究の目的でのみ提供されます。本プロジェクトの使用により生じたアカウントの停止、サービスの中断、データの損失、その他一切の直接的または間接的な損害について、作者は一切の責任を負いません。
- **🚫 商用利用の非許諾**：本プロジェクトの開発者は、いかなる個人または組織に対しても、本プロジェクトを利用したいかなる形態の商業運営も一切許諾していません。本プロジェクトの名義で、または本プロジェクトに基づいて行われる商業行為はすべて本プロジェクトおよびその開発者とは無関係であり、それにより生じる一切の紛争、損失、法的責任は行為者自身が負うものとします。

## 概要

Sub3API は個人利用と小規模チーム向けの AI API ゲートウェイおよび管理プラットフォームです。主流クライアント互換の API 中継と、リリース版・Docker イメージに組み込まれた管理画面を提供します。登録、決済、その他の SaaS 機能はデプロイ方法と設定に応じて有効化・無効化できます。

Sub3API は、個人利用と内部運用向けに、認証、上流アカウントのスケジューリング、利用記録、負荷分散、API 中継を提供します。

## 機能

- **マルチアカウント管理** - 複数の上流アカウントタイプ（OAuth、APIキー）をサポート
- **APIキー配布** - ユーザー向けの APIキーの生成と管理
- **精密な課金** - トークンレベルの使用量追跡とコスト計算
- **スマートスケジューリング** - スティッキーセッション付きのインテリジェントなアカウント選択
- **同時実行制御** - ユーザーごと・アカウントごとの同時実行数制限
- **レート制限** - 設定可能なリクエスト数およびトークンレート制限
- **内蔵決済システム** - 決済設定、注文処理、残高チャージを提供
- **管理ダッシュボード** - 監視・管理のための Web インターフェース
- **外部システム連携** - 外部システム（チケット管理など）を iframe 経由で管理ダッシュボードに埋め込み可能

## エコシステム

Sub3API を拡張・統合するコミュニティプロジェクト（現在はいずれも通常更新中）:

| プロジェクト | 説明 | 機能 | 更新状態 |
|---------|-------------|----------|----------|
| [Sub3ApiPay](https://github.com/touwaeriol/sub3apipay) | セルフサービス決済システム | 決済機能の拡張・統合ツール。Sub3API には内蔵決済機能もあります。[決済設定ガイド](docs/PAYMENT.md)をご参照ください | 通常更新 |
| [sub3api-mobile](https://github.com/ckken/sub3api-mobile) | モバイル管理コンソール | ユーザー管理、アカウント管理、監視ダッシュボード、マルチバックエンド切り替えが可能なクロスプラットフォームアプリ（iOS/Android/Web）。Expo + React Native で構築 | 通常更新 |

## 技術スタック

| コンポーネント | 技術 |
|-----------|------------|
| バックエンド | Go 1.26.10, Gin, Ent, Wire |
| データベース | PostgreSQL 15+、Docker Compose の既定値は `postgres:18-alpine` |
| キャッシュ/キュー | Redis 7+、Docker Compose の既定値は `redis:8-alpine` |

---

## Nginx リバースプロキシに関する注意

Sub3API（または CRS）を Nginx でリバースプロキシし、Codex CLI と組み合わせて使用する場合、Nginx の `http` ブロックに以下の設定を追加してください:

```nginx
underscores_in_headers on;
```

Nginx はデフォルトでアンダースコアを含むヘッダー（例: `session_id`）を破棄するため、マルチアカウント構成でのスティッキーセッションルーティングに支障をきたします。

---

## デプロイ

### 方法1: スクリプトによるインストール（推奨）

GitHub Releases からビルド済みバイナリをダウンロードするワンクリックインストールスクリプトです。バージョンタグを push すると、Release workflow が資産をビルド・検証し、GitHub Release と添付ファイルを自動公開します。

#### 前提条件

- Linux サーバー（amd64 または arm64）
- PostgreSQL 15+（インストール済みかつ稼働中。Docker Compose の既定値は 18-alpine）
- Redis 7+（インストール済みかつ稼働中。Docker Compose の既定値は 8-alpine）
- root 権限

#### インストール手順

```bash
curl -fsSL https://raw.githubusercontent.com/yuchen0x1/sub3api-relay/main/deploy/install.sh | sudo bash
```

スクリプトは以下を実行します:
1. システムアーキテクチャの検出
2. 最新リリースのダウンロード
3. バイナリを `/opt/sub3api` にインストール
4. systemd サービスの作成
5. システムユーザーと権限の設定

#### インストール後の作業

```bash
# 1. サービスを起動
sudo systemctl start sub3api

# 2. 起動時の自動起動を有効化
sudo systemctl enable sub3api

# 3. ローカル端末から、ループバック専用のセットアップウィザードへ SSH トンネルを作成
ssh -L 8080:127.0.0.1:8080 user@YOUR_SERVER_IP

# 4. ローカルブラウザで http://127.0.0.1:8080 を開く
```

セットアップウィザードでは以下の設定を行います:
- データベース設定
- Redis 設定
- 管理者アカウントの作成

#### アップグレード

**管理ダッシュボード**の左上にある**アップデートを確認**ボタンをクリックすることで、ダッシュボードから直接アップグレードできます。

Web インターフェースでは以下が可能です:
- 新しいバージョンの自動確認
- ワンクリックでのアップデートのダウンロードと適用
- 必要に応じたロールバック

#### よく使うコマンド

```bash
# ステータスを確認
sudo systemctl status sub3api

# ログを表示
sudo journalctl -u sub3api -f

# サービスを再起動
sudo systemctl restart sub3api

# アンインストール
curl -fsSL https://raw.githubusercontent.com/yuchen0x1/sub3api-relay/main/deploy/install.sh | sudo bash -s -- uninstall -y
```

---

### 方法2: Docker Compose（推奨）

PostgreSQL と Redis のコンテナを含む Docker Compose でデプロイします。

#### 前提条件

- Docker 20.10+
- Docker Compose v2+

#### クイックスタート（ワンクリックデプロイ）

自動デプロイスクリプトを使用して簡単にセットアップできます:

```bash
# デプロイ用ディレクトリを作成
mkdir -p sub3api-relay-deploy && cd sub3api-relay-deploy

# デプロイ準備スクリプトをダウンロードして実行
curl -fsSL https://raw.githubusercontent.com/yuchen0x1/sub3api-relay/main/deploy/docker-deploy.sh | bash

# サービスを起動
docker compose up -d

# ログを表示
docker compose logs -f sub3api
```

**スクリプトの動作内容:**
- `docker-compose.local.yml`（`docker-compose.yml` として保存）と `.env.example` をダウンロード
- セキュアな認証情報（JWT_SECRET、TOTP_ENCRYPTION_KEY、POSTGRES_PASSWORD）を自動生成
- Docker の初回起動ではデータベースと Redis の準備後に `/setup` を表示し、サイト所有者が管理者アカウントとパスワードを設定します（`admin/admin` の既定値は作成しません）
- 自動生成されたシークレットで `.env` ファイルを作成
- データディレクトリを作成（バックアップ・移行が容易なローカルディレクトリを使用）
- インフラ用シークレットと一度だけ使うセットアップトークンを所有者のみ読み取り可能な `.env` に保存

#### 手動デプロイ

手動でセットアップする場合:

```bash
# 1. リポジトリをクローン
git clone https://github.com/yuchen0x1/sub3api-relay.git
cd sub3api-relay/deploy

# 2. 環境設定ファイルをコピー
cp .env.example .env
chmod 600 .env

# 3. 設定を編集（セキュアなパスワードを生成）
nano .env
```

**`.env` の必須設定:**

```bash
# PostgreSQL パスワード（必須）
POSTGRES_PASSWORD=your_secure_password_here

# JWT シークレット（推奨 - 再起動後もユーザーのログイン状態を保持）
JWT_SECRET=your_jwt_secret_here

# TOTP 暗号化キー（推奨 - 再起動後も二要素認証を維持）
TOTP_ENCRYPTION_KEY=your_totp_key_here

# 初期管理者アカウント
# 初回起動後に /setup でサイト所有者のアカウントとパスワードを設定

# オプション: カスタムポート
SERVER_PORT=8080
```

**セキュアなシークレットの生成方法:**
```bash
# JWT_SECRET を生成
openssl rand -hex 32

# TOTP_ENCRYPTION_KEY を生成
openssl rand -hex 32

# POSTGRES_PASSWORD を生成
openssl rand -hex 32
```

```bash
# 4. データディレクトリを作成（ローカルバージョンの場合）
mkdir -p data postgres_data redis_data

# 5. すべてのサービスを起動
# オプション A: ローカルディレクトリバージョン（推奨 - 移行が容易）
docker compose -f docker-compose.local.yml up -d

# オプション B: 名前付きボリュームバージョン（シンプルなセットアップ）
docker compose up -d

# 6. ステータスを確認
docker compose -f docker-compose.local.yml ps

# 7. ログを表示
docker compose -f docker-compose.local.yml logs -f sub3api
```

#### デプロイバージョン

| バージョン | データストレージ | 移行 | 推奨用途 |
|---------|-------------|-----------|----------|
| **docker-compose.local.yml** | ローカルディレクトリ | ✅ 容易（ディレクトリ全体を tar） | 本番環境、頻繁なバックアップ |
| **docker-compose.yml** | 名前付きボリューム | ⚠️ docker コマンドが必要 | シンプルなセットアップ |

**推奨:** データ管理が容易な `docker-compose.local.yml`（スクリプトによるデプロイ）を使用してください。

#### アクセス

リバースプロキシで公開した HTTPS アドレスを開いてください。Compose は既定で `127.0.0.1:8080` のみにバインドするため、信頼できる TLS リバースプロキシまたは SSH トンネルを設定するまで直接公開されません。

初回起動後に `/setup` でサイト所有者のアカウントとパスワードを設定し、その後 `/login` からログインしてください。

#### アップグレード

```bash
# 対象リリースを固定し、イメージを取得してコンテナを再作成
sed -i.bak 's|^SUB3API_IMAGE=.*|SUB3API_IMAGE=ghcr.io/yuchen0x1/sub3api-relay:0.4.1|' .env
docker compose pull sub3api
docker compose up -d --no-deps sub3api
```

#### 簡単な移行（ローカルディレクトリバージョン）

`docker-compose.local.yml` を使用している場合、新しいサーバーへの移行が簡単です:

```bash
# 移行元サーバーにて
docker compose -f docker-compose.local.yml down
cd ..
tar czf sub3api-complete.tar.gz sub3api-deploy/

# 新しいサーバーに転送
scp sub3api-complete.tar.gz user@new-server:/path/

# 移行先サーバーにて
tar xzf sub3api-complete.tar.gz
cd sub3api-relay-deploy/
docker compose -f docker-compose.local.yml up -d
```

#### よく使うコマンド

```bash
# すべてのサービスを停止
docker compose -f docker-compose.local.yml down

# 再起動
docker compose -f docker-compose.local.yml restart

# すべてのログを表示
docker compose -f docker-compose.local.yml logs -f

# すべてのデータを削除（注意！）
docker compose -f docker-compose.local.yml down
rm -rf data/ postgres_data/ redis_data/
```

---

### 方法3: Apple container（macOS）

Apple シリコン搭載 Mac と macOS 26 では、Apple `container` 1.1.0 以降を使用して Sub3API、PostgreSQL、Redis の完全なスタックを実行できます:

```bash
git clone https://github.com/yuchen0x1/sub3api-relay.git
cd sub3api-relay/deploy
./apple-container.sh init
./apple-container.sh up
./apple-container.sh status
```

これはローカル開発および手動運用向けです。本番環境では引き続き Docker Compose を推奨します。ライフサイクル、永続化、アップグレード、制限については [deploy/APPLE_CONTAINER.md](deploy/APPLE_CONTAINER.md) を参照してください。

---

### 方法4: ソースからビルド

開発やカスタマイズのためにソースコードからビルドして実行します。

#### 前提条件

- Go 1.26.10
- Node.js 24
- pnpm 11.20.0
- PostgreSQL 15+（Docker Compose の既定値は 18-alpine）
- Redis 7+（Docker Compose の既定値は 8-alpine）

#### ビルド手順

```bash
# 1. リポジトリをクローン
git clone https://github.com/yuchen0x1/sub3api-relay.git
cd sub3api-relay

# 2. フロントエンド資産をインストールしてビルド
cd frontend
corepack enable
pnpm install --frozen-lockfile
pnpm build

# 3. フロントエンドを埋め込んだバックエンドをビルド
cd ../backend
VERSION="$(./scripts/resolve-version.sh)"
go build -tags embed -ldflags="-X main.Version=${VERSION}" -o sub3api ./cmd/server

# 4. 設定ファイルを作成
cp ../deploy/config.example.yaml ./config.yaml

# 5. 設定を編集
nano config.yaml
```


**`config.yaml` の主要設定:**

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "sub3api"

redis:
  host: "localhost"
  port: 6379
  password: ""

jwt:
  secret: "change-this-to-a-secure-random-string"
  expire_hour: 24

default:
  user_concurrency: 5
  user_balance: 0
  api_key_prefix: "sk-"
  rate_multiplier: 1.0
```

### Sora ステータス（一時的に利用不可）

> ⚠️ Sora 関連の機能は、上流統合およびメディア配信の技術的問題により一時的に利用できません。
> 現時点では本番環境で Sora に依存しないでください。
> 既存の `gateway.sora_*` 設定キーは予約されていますが、これらの問題が解決されるまで有効にならない場合があります。

`config.yaml` では追加のセキュリティ関連オプションも利用できます:

- `cors.allowed_origins` - CORS 許可リスト
- `security.response_headers.enabled` - 設定可能なレスポンスヘッダーフィルタリングを有効化（無効時はデフォルトの許可リストを使用）
- `security.csp` - Content-Security-Policy ヘッダーの制御
- `billing.circuit_breaker` - 課金エラー時にフェイルクローズ
- `security.trust_forwarded_ip_for_api_key_acl` - 従来の生転送ヘッダーによる上書きを制御（アップグレード互換性のため既定で有効）。無効にすると `server.trusted_proxies` を厳格に使用し、Sub3API に直接接続するプロキシの正確な CIDR のみを指定
- `security.forwarded_client_ip_headers` - サードパーティ CDN のクライアント IP ヘッダーを最大 16 個指定。従来モードが有効な場合のみ、設定順で組み込みヘッダーより先に評価
- `turnstile.required` - リリースモードでの Turnstile 必須化

カスタムクライアント IP ヘッダーは YAML またはカンマ区切りの環境変数で設定できます:

```bash
SECURITY_FORWARDED_CLIENT_IP_HEADERS=True-Client-IP,X-CDN-Client-IP
```

ヘッダー名は検証、正規化、大小文字を区別しない重複排除が行われます。管理画面のセキュリティ設定から再起動せずに更新でき、新規インストールでは YAML/環境変数の既定値を保存し、既存環境ではデータベース値がない場合に補完します。従来モードを無効にするとカスタムおよび組み込みの生転送ヘッダーはすべて無視され、`server.trusted_proxies` のみを使用します。有効にする場合はオリジンへの接続元を CDN/プロキシに制限し、エッジで信頼する全クライアント IP ヘッダーを上書きしてください。移行規則と信頼境界の詳細は [`deploy/EDGE_SECURITY.md`](deploy/EDGE_SECURITY.md) を参照してください。

**上流 Base URL について**

上流アドレスは有効な http/https URL であればよく、ドメイン許可リストやプライベートアドレスの制限はありません。公式 API・サードパーティ中継・自己ホスト型ゲートウェイ（`http://localhost` や内部 IP を含む）をそのまま設定できます。本番環境では API キーの平文送信や中間者攻撃を避けるため、HTTPS 上流を推奨します。

レスポンスヘッダーフィルタリングを無効にする場合は、ネットワーク層を強化してください:
- プライベート/ループバック/リンクローカル範囲をブロック
- TLS のみのアウトバウンドトラフィックを強制
- プロキシで機密性の高い上流レスポンスヘッダーを除去

#### ⚠️ 重要：管理者アカウントの作成

初期管理者アカウントは初回セットアップ時に作成されます。Docker で `AUTO_SETUP_OWNER_SETUP=true` を使用すると、インフラ準備後に `/setup` でサイト所有者が資格情報を設定します。`admin/admin` の既定値は作成されず、既存の管理者も上書きされません。

`config.yaml` を事前に作成した場合でも、インストール完了ロックがないときはセットアップ状態が維持されます。Docker の owner-setup モードでは、インフラ準備後に `/setup` が表示され、データベースと Redis の認証情報をブラウザへ入力せずにサイト所有者だけがアカウントを作成できます。

**管理者アカウントを作成する 2 つの方法:**

1. **Docker/自動デプロイ:** 起動後に `/setup` を開き、サイト所有者のログイン名、パスワード、確認パスワードだけを入力します。完了後にサービスが再起動し、`/login` からログインできます。

2. **すでに `config.yaml` を作成してしまった場合:** 初回起動前に一時的に退避してウィザードを発生させ、完了後に戻します:
   ```bash
   mv config.yaml config.yaml.bak
   ./sub3api        # ウィザードが http://localhost:8080 で起動し、新しい config.yaml を生成します
   # ウィザード完了後、Ctrl+C でサーバーを停止し、設定を復元します:
   mv config.yaml.bak config.yaml
   ./sub3api        # 通常モードで再起動し、作成した管理者でログインします
   ```

```bash
# 6. アプリケーションを実行
./sub3api
```

#### 開発モード

```bash
# バックエンド（ホットリロード付き）
cd backend
go run ./cmd/server

```

#### コード生成

`backend/ent/schema` を編集した場合、Ent + Wire を再生成してください:

```bash
cd backend
go generate ./ent
go generate ./cmd/server
```

---

## シンプルモード

シンプルモードは、フル SaaS 機能を必要とせず、素早くアクセスしたい個人開発者や社内チーム向けに設計されています。

- 有効化: 環境変数 `RUN_MODE=simple` を設定
- 違い: SaaS 関連機能を非表示にし、課金プロセスをスキップ
- セキュリティに関する注意: 本番環境では `SIMPLE_MODE_CONFIRM=true` も設定する必要があります

---

## Antigravity サポート

Sub3API は [Antigravity](https://antigravity.so/) アカウントをサポートしています。認証後、Claude および Gemini モデル用の専用エンドポイントが利用可能になります。

### 専用エンドポイント

| エンドポイント | モデル |
|----------|-------|
| `/antigravity/v1/messages` | Claude モデル |
| `/antigravity/v1beta/` | Gemini モデル |

### Claude Code の設定

```bash
export ANTHROPIC_BASE_URL="http://localhost:8080/antigravity"
export ANTHROPIC_AUTH_TOKEN="sk-xxx"
```

### ハイブリッドスケジューリングモード

Antigravity アカウントはオプションの**ハイブリッドスケジューリング**をサポートしています。有効にすると、汎用エンドポイント `/v1/messages` および `/v1beta/` も Antigravity アカウントにリクエストをルーティングします。

> **⚠️ 警告**: 異なるソースの Claude 互換アカウントは**同じ会話コンテキスト内で混在させることはできません**。グループを使用して適切に分離してください。

---

## プロジェクト構成

```
sub3api/
├── backend/                  # Go バックエンドサービス
│   ├── cmd/server/           # アプリケーションエントリ
│   ├── internal/             # 内部モジュール
│   │   ├── config/           # 設定
│   │   ├── model/            # データモデル
│   │   ├── service/          # ビジネスロジック
│   │   ├── handler/          # HTTP ハンドラー
│   │   └── gateway/          # API ゲートウェイコア
│   └── resources/            # 静的リソース
│
│   └── src/
│       ├── api/              # API 呼び出し
│       ├── stores/           # 状態管理
│       ├── views/            # ページコンポーネント
│       └── components/       # 再利用可能なコンポーネント
│
└── deploy/                   # デプロイファイル
    ├── docker-compose.yml    # Docker Compose 設定
    ├── .env.example          # Docker Compose 用環境変数
    ├── config.example.yaml   # バイナリデプロイ用フル設定ファイル
    └── install.sh            # ワンクリックインストールスクリプト
```

## スター履歴

<a href="https://star-history.com/#yuchen0x1/sub3api-relay&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=yuchen0x1/sub3api-relay&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=yuchen0x1/sub3api-relay&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=yuchen0x1/sub3api-relay&type=Date" />
 </picture>
</a>

---

## ライセンス

本プロジェクトは [GNU Lesser General Public License v3.0](LICENSE)（またはそれ以降のバージョン）の下でライセンスされています。

Copyright (c) 2026 Wesley Liddick

---

<div align="center">

**このプロジェクトが役に立ったら、ぜひスターをお願いします！**

</div>
