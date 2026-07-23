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
- `UserPromptSubmit`: 프롬프트를 제출할 때

두 훅은 비동기로 실행됩니다. 세션 시작 훅은 첫 spinner 항목을 준비하고, 프롬프트 제출 훅은 다음 요청에서 표시할 항목 하나를 준비합니다. Claude Code는 spinner를 훅보다 먼저 선택하므로, 제출 훅에서 갱신한 항목은 다음 요청부터 표시됩니다. 다음 프롬프트를 훅 완료 전 제출하면 이전 항목이 한 번 더 표시될 수 있습니다.

후보군은 최근 24시간 내 모든 글입니다. 이 기간의 글이 10개보다 적으면 최신순으로 이전 글을 추가해 최소 10개를 유지합니다. 각 항목은 `[10p] 제목 - 요약`과 터미널 링크를 포함합니다.

후보 ID와 마지막으로 준비한 항목은 설정 디렉터리의 `claude-geeknews-spinner/rotation-state.json`에 저장됩니다. 새 후보가 있으면 가장 최신 새 글을 먼저 표시합니다. 후보가 동일하면 최신 글부터 오래된 글 순으로 한 항목씩 순차 회전하고, 마지막 항목 뒤에는 다시 최신 글로 돌아갑니다.

매번 `spinnerVerbs`에는 선택된 항목 하나만 기록하므로 Claude Code의 무작위 선택으로 같은 제목이 연속 선택되는 문제를 피합니다. 이미 진행 중인 요청의 spinner 문구는 바뀌지 않습니다.

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
