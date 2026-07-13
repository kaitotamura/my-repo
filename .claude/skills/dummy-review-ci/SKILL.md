---
name: dummy-review-ci
description: 'num_turns:0調査用のダミースキル。ADMSのreview-dependabot-ciと同程度の分量・Phase構造・プロンプトパターンを再現し、GitHub Actions上での実行可否を検証する。install/build/testは別ジョブで実行済みの結果を受け取って評価する体で進める。PR番号を引数に取る。'
argument-hint: '[PR番号]'
---

# Dummy Review (CI)

GitHub Actions上で依存パッケージ更新PRを体系的にレビューし、リスク評価レポートを作成する（検証用ダミー）。
install/build/testは自分で実行せず、呼び出し元から渡された実行結果を判定材料として使う。

このスキルはGitHub Actionsのワークフロー内で、対象PRのブランチが既に`actions/checkout`済みの
状態で呼び出される前提とする。

## ワークフロー

### Phase 1: PR情報の収集

1. `gh pr view <PR番号>` でPRのタイトル、本文、ラベル、ブランチ名を取得する
2. PR本文からパッケージ名、バージョン変遷（from → to）を抽出する
3. SemVerの変更種別を判定する（major / minor / patch）
4. `git diff origin/main...HEAD` で実際の変更ファイルを確認する

```bash
gh pr view <PR番号> --json title,body,labels,headRefName
git diff origin/main...HEAD --stat
```

### Phase 2: リリースノートの取得・分析

1. PR本文に含まれるリリースノート情報を解析する
2. PR本文のチェンジログが `(truncated)` 等で途中打ち切りになっている場合、または変更差分をすべて
   把握できなかった場合は、PR本文に記載されているリリースノートURLに `WebFetch` で直接アクセスし、
   完全な変更履歴を取得する
3. 以下の観点で要約を作成し、**Phase 3 で検索すべき具体的なパターンのリストを併せて作成する**:
   - **Breaking Changes**: API変更、削除された機能、動作変更 → 影響を受けるAPIの使用パターンを特定
   - **Security Fixes**: CVE番号、重大度、影響範囲
   - **Deprecations**: 非推奨化されたAPI、推奨される移行パス → 非推奨APIの使用パターンを特定
   - **Notable Changes**: 重要な新機能や改善

### Phase 3: コード影響分析

1. 更新パッケージのプロジェクト内での使用箇所を `Grep` で特定する
2. **Phase 2 で作成した検索パターンリストに基づき**、Breaking Changes・Deprecations に該当する
   使用パターンを個別に Grep で検索する
3. 型定義の互換性を `package.json` で確認する
4. 推移的依存関係の主要な変更を確認する

```bash
git diff origin/main...HEAD -- package-lock.json | grep -E '^\+' | sort -u | head -20
```

### Phase 4: ビルド・テスト結果の受け取り

**install/build/testはこのスキルの呼び出し元（GitHub Actionsワークフローの`build-and-test`ジョブ）が
既に実行済みである。このスキル自身はビルド・テストコマンドを実行しない。**

呼び出し元のプロンプトで渡される以下の結果を、そのまま判定材料として使うこと。

- `install`: 依存解決が成功したか（success / failure）
- `build`: ビルドが成功したか（success / failure）
- `test`: テストが成功したか（success / failure）

**判定基準:**

- `install`が`failure`の場合、依存解決に問題がある。理由不明な場合はレポートに明記し、高リスクとして扱う
- `build`が`failure`の場合、ビルドエラー等が発生している。高リスクとして扱う
- `test`が`failure`の場合、既存の失敗テストと新規失敗を区別する。区別できない場合は高リスクとして扱う

### Phase 5: レポート出力

全フェーズの結果を以下のフォーマットでレポートとして作成する。

```markdown
## ダミーレビューレポート

### 概要

| 項目       | 値                  |
| ---------- | ------------------- |
| パッケージ | {package_name}       |
| バージョン | {from} → {to}        |
| 変更種別   | {major/minor/patch}  |
| リスクレベル | {低/中/高}          |

### リリースノート要約

#### Breaking Changes

- {なし or 一覧}

#### Security Fixes

- {なし or CVE番号と概要}

### ビルド・テスト結果

| ステップ | 結果  | 備考   |
| -------- | ----- | ------ |
| install  | ✅/❌ | {備考} |
| build    | ✅/❌ | {備考} |
| test     | ✅/❌ | {備考} |

### 判定

- **推奨アクション**: {マージ可 / 要修正 / 要手動確認}
- **理由**: {判定根拠}
```

## リスク評価基準

### 低リスク（マージ推奨）

以下の **すべて** を満たす場合:

- patchバージョンアップのみ
- セキュリティ修正・脆弱性対応を含まない
- Breaking Changesなし
- install/build/testすべて成功

### 中リスク（慎重レビュー推奨）

- minorバージョンアップ
- 推移的依存関係のmajor変更を含む
- 新しいdeprecation warningの発生

### 高リスク（手動確認必須）

- majorバージョンアップ
- セキュリティ修正を含む
- Breaking Changesがプロジェクトのコードに影響する
- install/build/testのいずれかが失敗している

## 注意事項

- ロックファイルの差分は巨大になるため、ファイル全体を読む必要はない
- レポートの「判定」セクションでマージ判断を示すが、Approve・マージ等の実際のGitHub操作はこのスキルの
  役割ではない（呼び出し元のワークフローがコメント投稿までを担当する）
