# golang-repo-template
This project is a template repository for my typical golang repositories

## Setting Versioning Information

```bash
go build -ldflags "\
  -X 'github.com/bgrewell/stencil.appVersion=v1.2.3' \
  -X 'github.com/bgrewell/stencil.appBuildDate=2025-07-23' \
  -X 'github.com/bgrewell/stencil.appCommitHash=abc1234' \
  -X 'github.com/bgrewell/stencil.appBranch=main'"
```