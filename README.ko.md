# Claude GeekNews Spinner

Claude Code가 작업하는 동안 [GeekNews 최신글](https://news.hada.io/new) 제목을 스피너에 표시합니다.

```text
[GN] 새로운 오픈 소스 데이터베이스 엔진
[GN] 빌드 시간을 80퍼센트 줄인 방법
[GN] 코딩 에이전트 실전 가이드
```

기본값은 GeekNews 최신글 10개입니다. 항목 수, 갱신 주기, 접두어, 제목 길이, 소스 URL, 표시 위치를 변경할 수 있습니다.

[English documentation](README.md)

## 실시간에 가까운 갱신 방식

Claude Code는 URL을 스피너 데이터로 직접 읽을 수 없습니다. 이 도구는 가벼운 비동기 훅 두 개를 설치합니다.

- `SessionStart`는 세션 시작과 재개 시 최신 제목을 확인합니다.
- `UserPromptSubmit`은 사용자가 프롬프트를 보낼 때 다시 확인합니다.

기본 캐시 유효 시간은 GeekNews 최신글 페이지의 공개 캐시 정책과 같은 15초입니다. 캐시가 최신이면 네트워크 요청 없이 바로 종료합니다. 캐시가 오래됐으면 백그라운드에서 갱신하므로 Claude Code 작업을 기다리게 하지 않습니다. 실행 중인 세션도 변경된 설정을 자동으로 읽습니다.

daemon을 사용하지 않으며 훅 실행 사이에 상주하는 프로세스도 없습니다.

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
  --interval 30s \
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
claude-geeknews-spinner config set interval 1m
claude-geeknews-spinner config set display tip
```

기본 설정:

```json
{
  "count": 10,
  "refreshInterval": "15s",
  "sourceUrl": "https://news.hada.io/new",
  "prefix": "[GN] ",
  "maxTitleRunes": 100,
  "displayMode": "verb"
}
```

| 설정 | 값 | 설명 |
| --- | --- | --- |
| `count` | 1에서 50 | 최신 제목 개수입니다. 필요하면 다음 페이지도 읽습니다. |
| `refreshInterval` | 15초에서 24시간 | 다음 네트워크 요청까지의 최소 시간입니다. |
| `sourceUrl` | HTTP 또는 HTTPS 절대 URL | GeekNews HTML 구조와 Atom 피드를 지원합니다. |
| `prefix` | 문자열 | 각 제목 앞에 표시할 문구입니다. |
| `maxTitleRunes` | 20에서 500 | 제목을 줄이기 전 최대 문자 수입니다. |
| `displayMode` | `verb`, `tip`, `both` | 제목이 표시될 위치입니다. |

`verb`는 제목을 메인 스피너 문구에 넣습니다. Claude Code가 작업 완료 문구에도 제목을 사용할 수 있습니다. `tip`은 보조 팁 영역을 사용합니다. `both`는 두 위치에 모두 기록합니다.

## 명령

```text
claude-geeknews-spinner install [options]
claude-geeknews-spinner refresh
claude-geeknews-spinner config [show|path|set <key> <value>]
claude-geeknews-spinner status
claude-geeknews-spinner uninstall [--purge]
```

`refresh`는 즉시 네트워크 요청을 실행합니다. `status`는 설치 상태, 설정, 캐시 개수, 마지막 성공 시간을 표시합니다. `uninstall`은 이 도구가 설치한 훅만 제거하고 설치 전 스피너 값을 복원합니다. `--purge`를 추가하면 설정과 캐시도 삭제합니다.

## 안전성

- 기존 Claude Code 설정과 관련 없는 훅을 보존합니다.
- 잠금과 원자적 파일 교체를 사용합니다.
- 잘못된 Claude 설정을 빈 객체로 덮어쓰지 않습니다.
- 네트워크 요청이 실패하거나 결과가 비어 있으면 마지막 정상 캐시를 유지합니다.
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
