# docs

プロジェクトのドキュメント置き場。

## 構成

| パス | 内容 |
| --- | --- |
| `engineering/` | 開発に必要な設計(アーキテクチャ、データ設計、API、運用など)をまとめる |
| `operations/` | 運用ルール(GitHub の利用フローなど)と、リリース手順、削除・作り直しからの復旧手順、リリースゲート(リリース PR に出す、マイルストーンの Issue の確認) |
| `infra/` | インフラ構成図(`architecture.png`)と、その生成スクリプト(`generate_architecture.py`)・アイコン |
| `mocks/` | Flex Message の見た目や、実際に届く表示イメージを全シナリオ分まとめた HTML モック |

## ルール

- 設計は Markdown で `engineering/` に置く
- 新しい Flex Message やシナリオを追加したら `mocks/` に HTML モックも追加する
- 設計とモックは実装と食い違わないよう、変更時に同時に更新する
