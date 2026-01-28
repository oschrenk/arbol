# Examples

`arbol status` outputs JSON by default. Pipe it into `jq` to filter, transform, and query your repositories.

## List dirty or diverged repos

```bash
arbol status | jq '[.[] | select(.changes.dirty or .remote.diverged)]'
```

## Get repos not yet cloned

```bash
arbol status | jq '[.[] | select(.branch == null)]'
```

## Count dirty files across all repos

```bash
arbol status | jq '[.[].changes.files // 0] | add'
```

## Sort by latest commit, uncloned last

```bash
arbol status | jq 'sort_by(.changes.last_commit // "") | reverse'
```

## Table output with `--plain`

```
$ arbol status --plain
PATH                            BRANCH           WORK   REMOTE    AGE     COMMENTS
work.backend.api                main             ✔      ✔         2d
work.backend.worker             feature/auth     ● 3    ↑2        3d      3 dirty files, 2 unpushed commits
personal.dotfiles               main             ✔      ↓5        1w      5 commits behind origin
personal.golang.arbol           main             ✔      ↓2 ↑1     5d      diverged
external.archived               ab5c2d0          ✔      ✔         2mo     detached HEAD
```
