# Claude GeekNews Spinner

Claude Code가 작업하는 동안 [GeekNews 최신글](https://news.hada.io/new) 제목을 스피너에 표시합니다.

```text
[GN] 새로운 오픈 소스 데이터베이스 엔진
[GN] 빌드 시간을 80퍼센트 줄인 방법
[GN] 코딩 에이전트 실전 가이드
```

기본값은 GeekNews 최신글 10개입니다. 항목 수, 접두어, 제목 길이, 소스 URL, 표시 위치, 실험적 터미널 링크를 변경할 수 있습니다.

[English documentation](README.md)

## 실시간에 가까운 갱신 방식

Claude Code는 URL을 스피너 데이터로 직접 읽을 수 없습니다. 이 도구는 가벼운 비동기 훅 두 개를 설치합니다.

- `SessionStart`는 세션 시작과 재개 시 최신 제목을 가져옵니다.
- `UserPromptSubmit`은 사용자가 프롬프트를 보낼 때 다시 가져옵니다.

각 훅은 실시간 네트워크 요청을 수행합니다. 비동기 훅이므로 Claude Code는 네트워크 응답을 기다리지 않으며, 성공한 결과는 이후 스피너 선택에 사용됩니다. 제목을 영구 캐시하지 않습니다.

daemon을 사용하지 않으며 훅 실행 사이에 상주하는 프로세스도 없습니다.

`spinnerVerbs`는 시간에 따라 순환하는 재생 목록이 아니라 선택 후보 목록입니다. Claude Code는 일반적으로 turn마다 항목 하나를 선택하고 turn이 초기화될 때까지 유지합니다. 내장 thinking 진행 상태, 도구 실행, task 상태가 화면의 문구를 일시적으로 바꿀 수는 있지만 custom verb 자체를 시간 간격으로 순환하지는 않습니다. `UserPromptSubmit` 훅은 비동기로 실행되므로 새로 가져온 목록은 훅을 발생시킨 현재 turn보다 이후 선택에 확실히 반영됩니다.

## 설치

### Go로 설치

Go 1.21 이상이 필요합니다.

```bash
go install github.com/saz/claude-geeknews-spinner/cmd/claude-geeknews-spinner@latest
claude-geeknews-spinner install
```

버전 태그를 push하면 Linux, macOS, Windows용 빌드 파일이 GitHub Releases에 게시됩니다.

### 설치하면서 설정 변경

```bash
claude-geeknews-spinner install \
  --count 20 \
  --display verb \
  --prefix "[GeekNews] " \
  --max-title-runes 120
```

## 설정

현재 설정과 설정 파일 경로를 표시합니다.

```bash
claude-geeknews-spinner config
claude-geeknews-spinner config path
```

설정을 변경하면 현재 스피너도 즉시 갱신됩니다.

```bash
claude-geeknews-spinner config set count 20
claude-geeknews-spinner config set display tip
claude-geeknews-spinner config set clickable-links true
```

기본 설정:

```json
{
  "count": 10,
  "sourceUrl": "https://news.hada.io/new",
  "prefix": "[GN] ",
  "maxTitleRunes": 100,
  "displayMode": "verb",
  "clickableLinks": false
}
```

| 설정 | 값 | 설명 |
| --- | --- | --- |
| `count` | 1에서 50 | 최신 제목 개수입니다. 필요하면 다음 페이지도 읽습니다. |
| `sourceUrl` | HTTP 또는 HTTPS 절대 URL | GeekNews HTML 구조와 Atom 피드를 지원합니다. |
| `prefix` | 문자열 | 각 제목 앞에 표시할 문구입니다. |
| `maxTitleRunes` | 20에서 500 | 제목을 줄이기 전 최대 문자 수입니다. |
| `displayMode` | `verb`, `tip`, `both` | 제목이 표시될 위치입니다. |
| `clickableLinks` | `true`, `false` | 제목에 실험적인 OSC 8 터미널 링크를 적용합니다. |

`verb`는 Claude Code 기본 스피너 verb를 유지하면서 GeekNews 제목을 뒤에 추가합니다. Claude Code가 작업 완료 문구에도 해당 문구를 사용할 수 있습니다. 각 항목은 GeekNews가 제공하는 경우 제목과 본문 요약 첫 줄을 함께 표시합니다. `tip`은 보조 팁 영역을 사용합니다. `both`는 두 위치에 모두 기록합니다.

`clickableLinks`를 활성화하면 지원 터미널에서 macOS는 Cmd+클릭, Linux와 Windows는 Ctrl+클릭으로 제목을 열 수 있습니다. Claude Code는 `spinnerVerbs`의 OSC 8 지원을 명시하지 않으므로 지원하지 않는 렌더러는 일반 텍스트로 표시하거나 링크를 제거할 수 있습니다.

## 명령

```text
claude-geeknews-spinner install [options]
claude-geeknews-spinner refresh
claude-geeknews-spinner config [show|path|set <key> <value>]
claude-geeknews-spinner status
claude-geeknews-spinner uninstall [--purge]
```

`refresh`는 즉시 네트워크 요청을 실행합니다. `status`는 설치 상태와 설정 경로를 표시합니다. `uninstall`은 이 도구가 설치한 훅만 제거하고 설치 전 스피너 값을 복원합니다. `--purge`를 추가하면 설정도 삭제합니다.

## 안전성

- 기존 Claude Code 설정과 관련 없는 훅을 보존합니다.
- 잠금과 원자적 파일 교체를 사용합니다.
- 잘못된 Claude 설정을 빈 객체로 덮어쓰지 않습니다.
- 네트워크 요청이 실패하거나 결과가 비어 있으면 현재 적용된 스피너 설정을 변경하지 않습니다.
- 설치 전 스피너 값을 저장하고 제거할 때 복원합니다.
- 관리 중인 스피너 키를 사용자가 직접 바꾸면 덮어쓰지 않고 변경을 감지합니다.
- 원격 제목의 제어 문자와 양방향 텍스트 제어 문자를 제거합니다.
- 다른 Claude Code 프로필을 위한 `CLAUDE_CONFIG_DIR`를 지원합니다.

설정된 소스 URL 외에는 네트워크 요청을 보내지 않으며 telemetry를 수집하지 않습니다.

## 개발

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/claude-geeknews-spinner
```

기여 방법은 [CONTRIBUTING.md](CONTRIBUTING.md)를 참고하십시오.
[설계 문서](docs/design.md)에는 관련 spinner 프로젝트와 갱신 구조를 비교한 결과가 있습니다.

## 라이선스

MIT
