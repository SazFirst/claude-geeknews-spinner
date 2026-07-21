# Claude GeekNews Spinner

Claude Code 스피너에 GeekNews 최신글을 표시하는 플러그인입니다.

사용자용 CLI나 앱 설정은 없습니다. 플러그인을 활성화하면 비동기 `SessionStart`, `UserPromptSubmit` 훅이 자동으로 등록됩니다.

## 설치

이 저장소를 Git 호스트에 게시한 뒤 Claude Code에서 마켓플레이스를 추가하고 플러그인을 설치합니다.

```text
/plugin marketplace add <owner>/<repository>
/plugin install claude-geeknews-spinner@geeknews-spinner
```

자체 호스팅 Git 저장소는 첫 명령에 전체 Git URL을 사용합니다. `.claude-plugin/marketplace.json`이 포함된 플러그인 디렉터리를 가리킵니다. 매니페스트에 고정 버전을 두지 않았으므로 push된 Git 커밋은 `/plugin update`로 갱신할 수 있습니다.

플러그인에 포함된 갱신 스크립트를 실행하려면 Node.js 18 이상이 필요합니다.

로컬 개발 시에는 플러그인 디렉터리를 직접 로드합니다.

```bash
claude --plugin-dir ./plugins/claude-geeknews-spinner
```

## 동작

세션 시작과 프롬프트 제출 때마다 훅이 백그라운드에서 GeekNews 최신글 페이지를 가져옵니다. 사용할 수 있는 첫 10개 글의 제목과 요약을 결합해 터미널 링크와 함께 `spinnerVerbs`에 기록합니다.

갱신은 Claude Code를 기다리게 하지 않습니다. 네트워크 또는 파싱 오류가 나면 기존 스피너 값은 유지됩니다. 동시에 여러 갱신이 실행되면 마지막 작성 결과가 남습니다.

플러그인을 제거하면 훅도 제거됩니다. 마지막으로 기록된 스피너 값은 다른 도구나 수동 편집으로 바꾸기 전까지 Claude 설정에 남습니다.

## 개발

```bash
node --test plugins/claude-geeknews-spinner/scripts/*.test.mjs
node --check plugins/claude-geeknews-spinner/scripts/refresh.mjs
claude plugin validate .
```

기여 방법은 [CONTRIBUTING.md](CONTRIBUTING.md)를 참고하십시오.

## 라이선스

MIT
