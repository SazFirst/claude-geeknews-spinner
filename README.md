# Claude GeekNews Spinner

Claude Code가 작업하는 동안 스피너에 GeekNews 최신글을 표시하는 플러그인입니다.

별도의 CLI나 설정은 필요하지 않습니다.

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

설치하면 플러그인의 `hooks/hooks.json`이 다음 비동기 훅을 등록합니다.

- `SessionStart`: Claude Code 세션을 시작할 때
- `UserPromptSubmit`: 프롬프트를 제출할 때

두 훅은 `scripts/refresh.mjs`를 백그라운드에서 실행합니다. 이 스크립트는 GeekNews 최신글 페이지에서 최근 10개 글의 제목과 요약을 가져와 터미널 링크 형식의 스피너 문구로 만듭니다.

생성한 문구는 Claude Code 설정의 `spinnerVerbs`를 교체합니다. 따라서 Claude Code가 작업 중일 때 기본 스피너 문구 대신 GeekNews 항목 중 하나가 표시됩니다.

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