# Claude GeekNews Spinner

Claude Code가 작업하는 동안 스피너에 GeekNews 최신글을 표시하는 플러그인입니다.

![Claude Code 스피너에 표시된 GeekNews 최신글](example.png)

## 설치

Claude Code에서 아래 명령을 순서대로 실행합니다.

```text
/plugin marketplace add SazFirst/claude-geeknews-spinner
/plugin install claude-geeknews-spinner@geeknews-spinner
```

설치 후 새 세션을 열면 자동으로 동작합니다.

## 업데이트

새 버전은 다음 명령으로 적용합니다.

```text
/plugin update claude-geeknews-spinner@geeknews-spinner
```

## 동작

설치하면 플러그인의 `hooks/hooks.json`이 다음 훅을 등록합니다.

- `SessionStart`: Claude Code 세션을 시작할 때
- `UserPromptSubmit`: 프롬프트를 제출할 때 rotator를 시작
- `Stop`: Claude의 응답이 끝날 때 rotator를 중지

`SessionStart` 훅은 `scripts/refresh.mjs`를 백그라운드에서 실행해 최신 제목 목록을 준비합니다. `UserPromptSubmit` 훅은 `scripts/rotate.mjs`를 백그라운드에서 시작합니다. rotator는 응답이 진행되는 동안 즉시 한 번, 이후 20초마다 GeekNews 최신글 페이지를 다시 가져옵니다.

후보군은 최근 24시간 내 모든 글입니다. 이 기간의 글이 10개보다 적으면 최신순으로 이전 글을 추가해 최소 10개를 유지합니다. 각 갱신에서는 후보군 중 순서대로 하나를 골라 `[10p] 제목 - 요약`과 터미널 링크를 포함한 하나의 `spinnerVerbs` 값으로 교체합니다. 따라서 Claude Code가 작업 중일 때 기본 스피너 문구 대신 GeekNews 항목이 20초 간격으로 바뀝니다. `Stop` 훅은 응답이 끝난 뒤 다음 주기에 rotator가 종료되도록 합니다.

설정 파일은 다음 순서로 수정됩니다.

- `CLAUDE_CONFIG_DIR`가 설정된 경우: `$CLAUDE_CONFIG_DIR/settings.json`
- 설정되지 않은 경우: `~/.claude/settings.json`

## 제거

```text
/plugin uninstall claude-geeknews-spinner@geeknews-spinner
```

플러그인을 제거하면 훅과 자동 갱신이 중지됩니다. 이미 기록된 `spinnerVerbs` 값은 Claude Code 설정 파일에 남습니다.

기본 스피너 문구로 되돌리려면 앞서 안내한 Claude Code 설정 파일을 열어 최상위 `spinnerVerbs` 항목 전체를 삭제한 뒤 Claude Code를 새로 시작합니다.

```json
{
  "spinnerVerbs": {
    "mode": "replace",
    "verbs": ["..."]
  }
}
```
