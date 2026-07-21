# Claude GeekNews Spinner

Claude Code 스피너에 GeekNews 최신글을 표시하는 플러그인입니다.

사용자용 CLI나 앱 설정은 없습니다. 플러그인을 활성화하면 비동기 `SessionStart`, `UserPromptSubmit` 훅이 자동으로 등록됩니다.

## 설치

Claude Code에서 아래 명령을 순서대로 실행합니다.

```text
/plugin marketplace add SazFirst/claude-geeknews-spinner
/plugin install claude-geeknews-spinner@geeknews-spinner
```

설치 후 새 세션을 열면 자동으로 동작합니다. 업데이트가 있으면 다음 명령으로 적용합니다.

```text
/plugin update claude-geeknews-spinner@geeknews-spinner
```

## 동작

세션 시작과 프롬프트 제출 때마다 훅이 백그라운드에서 GeekNews 최신글 페이지를 가져옵니다. 사용할 수 있는 첫 10개 글의 제목과 요약을 결합해 터미널 링크와 함께 `spinnerVerbs`를 대체합니다.

갱신은 Claude Code를 기다리게 하지 않습니다. 네트워크 또는 파싱 오류가 나면 기존 스피너 값은 유지됩니다. 동시에 여러 갱신이 실행되면 마지막 작성 결과가 남습니다.

플러그인을 제거하면 훅도 제거됩니다. 마지막으로 기록된 스피너 값은 다른 도구나 수동 편집으로 바꾸기 전까지 Claude 설정에 남습니다.
