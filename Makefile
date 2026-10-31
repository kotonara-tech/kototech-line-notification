# ルートからは backend の同名ターゲットを呼ぶ

.DEFAULT_GOAL := help

.PHONY: help test lint integration-test build clean verify verify-terraform

help: ## コマンド一覧を表示する
	@$(MAKE) -C backend help
	@grep -E '^verify:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

test lint integration-test build clean:
	@$(MAKE) -C backend $@

# 作業の完了前に実行する。既存の test / lint / build / terraform の検査を呼び、ドキュメントと yml は scripts/verify.mjs で見る
verify: ## 作業の完了前の検証(test・lint・build・Terraform・ドキュメントと yml の検査)
	$(MAKE) -C backend test lint build
	$(MAKE) verify-terraform
	node scripts/verify.mjs

verify-terraform:
	terraform fmt -check -recursive infra/terraform
	@for d in bootstrap env/dev env/prod; do \
		terraform -chdir=infra/terraform/$$d init -backend=false -input=false >/dev/null && \
		terraform -chdir=infra/terraform/$$d validate || exit 1; \
	done
