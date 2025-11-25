# docker-bootapp

멀티 프로젝트 Docker 네트워킹을 쉽게 관리하는 Docker CLI 플러그인.

자동으로 관리:
- 프로젝트별 고유 서브넷 할당 (충돌 방지)
- DOMAIN 설정이 있는 컨테이너의 /etc/hosts 항목
- **SSL 인증서** `SSL_DOMAINS` 도메인 자동 생성 및 시스템 신뢰 등록
- macOS 스마트 라우트 설정 (연결 테스트 후 라우트 추가)

## 설치

### 방법 1: Homebrew 사용 (macOS)

```bash
brew install yejune/tap/docker-bootapp
```

Homebrew가 자동으로:
- 최신 버전 다운로드 및 빌드
- Docker CLI 플러그인 설치 (`docker bootapp`)
- 독립 실행 파일 설치 (`bootapp`)
- 의존성 확인

### 방법 2: 설치 스크립트 사용 (Linux/macOS)

```bash
# 저장소 클론
git clone https://github.com/yejune/docker-bootapp.git
cd docker-bootapp

# 설치 스크립트 실행
bash install.sh
```

설치 스크립트가 자동으로:
- Go와 Docker 확인
- 바이너리 빌드
- `~/.docker/cli-plugins/docker-bootapp`에 설치 (Docker 플러그인)
- `/usr/local/bin/bootapp`에 설치 (독립 실행 파일)
- 플랫폼별 의존성 확인

### 방법 3: go install 사용
```bash
go install github.com/yejune/docker-bootapp@latest
docker-bootapp install
```

또는 로컬에서 빌드:
```bash
git clone https://github.com/yejune/docker-bootapp.git
cd docker-bootapp
go build
./docker-bootapp install
```

`install` 명령어가 자동으로:
- `~/.docker/cli-plugins/docker-bootapp`에 바이너리 복사
- `/usr/local/bin/bootapp`에 독립 실행 파일 설치
- 실행 권한 설정
- 플랫폼 의존성 확인

### 방법 4: 수동 설치
```bash
make build
cp build/docker-bootapp ~/.docker/cli-plugins/docker-bootapp
chmod +x ~/.docker/cli-plugins/docker-bootapp
sudo cp build/docker-bootapp /usr/local/bin/bootapp
sudo chmod +x /usr/local/bin/bootapp
```

## 사용법

### 컨테이너 시작
```bash
# docker-compose.yml 자동 감지
docker bootapp up

# compose 파일 지정
docker bootapp -f docker-compose.local.yml up
```

여러 compose 파일이 있으면 인터랙티브 선택 화면이 표시됩니다:
```
Select compose file:
▸ docker-compose.yml
  docker-compose.local.yml
  docker-compose.prod.yml

↑/↓ 화살표로 이동, Enter로 선택
```

지원하는 파일 패턴:
- `docker-compose.yml`, `docker-compose.yaml`
- `docker-compose.*.yml`, `docker-compose.*.yaml` (예: docker-compose.local.yml)
- `compose.yml`, `compose.yaml`

옵션:
- `-d, --detach`: 백그라운드 실행 (기본값: true)
- `--no-build`: 이미지 빌드 안 함
- `--pull`: 시작 전 이미지 pull
- `-F, --force-recreate`: 컨테이너 강제 재생성 + SSL 인증서 재생성

실행 시:
1. 프로젝트에 고유 서브넷 할당 (172.18-31.x.x 범위)
2. docker-compose 파일에서 DOMAIN/SSL_DOMAINS 설정 파싱
3. **SSL 인증서 생성** `SSL_DOMAINS` 도메인용 (없는 경우)
4. **시스템 trust store에 인증서 설치** (macOS Keychain / Linux ca-certificates)
5. docker-compose up으로 컨테이너 시작
6. 기본 compose 네트워크에서 컨테이너 IP 감지
7. 도메인 설정이 있는 컨테이너를 /etc/hosts에 추가
8. 필요 시 라우팅 설정 (macOS)

### 컨테이너 중지
```bash
docker bootapp down
docker bootapp -f docker-compose.local.yml down
```

옵션:
- `-v, --volumes`: 볼륨 삭제
- `--remove-orphans`: 고아 컨테이너 삭제
- `--keep-hosts`: /etc/hosts 항목 유지
- `--remove-config`: 글로벌 설정에서 프로젝트 삭제

### 프로젝트 목록
```bash
docker bootapp ls
```

## 도메인 설정

### 지원하는 환경변수

다음 환경변수들이 모두 사용됩니다 (중복 제거, 각각 단일/콤마/줄바꿈 구분 지원):
- `DOMAIN`
- `DOMAINS`
- `SSL_DOMAINS`
- `APP_DOMAIN`
- `VIRTUAL_HOST` (nginx-proxy 호환)

```yaml
services:
  app:
    image: nginx
    environment:
      SSL_DOMAINS: |
        myapp.local
        www.myapp.local

  mysql:
    image: mysql:8
    environment:
      DOMAIN: mysql.myapp.local

  redis:
    image: redis
    # DOMAIN 없음 = /etc/hosts에 추가 안됨 (IP만)
```

### Traefik 라벨

Traefik 라우터 규칙도 지원:

```yaml
services:
  web:
    image: nginx
    labels:
      - "traefik.http.routers.web.rule=Host(`web.local`)"
      # 여러 호스트 지원:
      - "traefik.http.routers.api.rule=Host(`api.local`) || Host(`api2.local`)"
      # 또는 콤마로 구분:
      - "traefik.http.routers.app.rule=Host(`app.local`, `www.app.local`)"
```

### 결과

도메인 설정이 있는 서비스만 /etc/hosts에 추가됩니다:

```
172.18.0.2    myapp.local        ## docker-bootapp:myproject
172.18.0.2    www.myapp.local    ## docker-bootapp:myproject
172.18.0.3    mysql.myapp.local  ## docker-bootapp:myproject
```

DOMAIN 설정이 없는 서비스(위의 redis)는 /etc/hosts에 추가되지 않습니다.

## SSL 인증서

### 자동 생성

bootapp은 `SSL_DOMAINS`에 지정된 도메인에 대해 자체 서명 SSL 인증서를 자동 생성합니다:

```yaml
services:
  app:
    image: nginx
    environment:
      SSL_DOMAINS: myapp.test
    volumes:
      - ./var/certs:/etc/nginx/certs:ro
```

인증서 특징:
- `./var/certs/` 디렉토리에 생성 (`.crt`, `.key`, `.pem` 파일)
- 시스템 키체인(macOS) 또는 ca-certificates(Linux)에 자동 신뢰 등록
- 10년 유효기간
- 브라우저 호환을 위한 SAN (Subject Alternative Name) 포함

### 인증서 파일

```
var/certs/
├── myapp.test.crt    # 인증서
├── myapp.test.key    # 개인 키
└── myapp.test.pem    # 인증서 + 키 결합
```

### 강제 재생성

인증서를 삭제하고 재생성하려면:

```bash
docker bootapp -f docker-compose.local.yml up -F
```

`-F` 플래그는:
1. trust store에서 기존 인증서 제거
2. 로컬 인증서 파일 삭제
3. 새 인증서 생성
4. trust store에 설치
5. 컨테이너 강제 재생성

### nginx 설정 예시

```nginx
server {
    listen 443 ssl;
    server_name myapp.test;

    ssl_certificate /etc/nginx/certs/myapp.test.crt;
    ssl_certificate_key /etc/nginx/certs/myapp.test.key;

    # ... 나머지 설정
}
```

## macOS 네트워킹

macOS에서 컨테이너 IP에 직접 접근하려면 **docker-mac-net-connect가 필수**입니다.

Docker Desktop은 Linux VM 안에서 컨테이너를 실행하므로, 네트워크 터널 없이는 macOS에서 컨테이너 IP에 직접 접근할 수 없습니다.

### 설치
```bash
brew install chipmk/tap/docker-mac-net-connect
sudo brew services start docker-mac-net-connect
```

bootapp은 docker-mac-net-connect를 확인하고, 없으면 설치 안내를 표시합니다.

## Linux

✅ **모든 기능이 Linux에서 네이티브로 동작합니다!**

Linux 지원 기능:

1. **Docker 네트워킹**
   - 컨테이너별 고유 subnet 할당
   - Container IP 직접 접근 (추가 도구 불필요)

2. **SSL 인증서 자동 생성 및 Trust**
   - Debian/Ubuntu: `update-ca-certificates`
   - RHEL/CentOS: `update-ca-trust`
   - 자체 서명 인증서를 시스템 trust store에 자동 등록

3. **/etc/hosts 자동 관리**
   - 도메인 → Container IP 매핑
   - 프로젝트별 자동 등록/제거

4. **독립 실행 파일 + Docker 플러그인**
   - `bootapp` 또는 `docker bootapp` 명령어 모두 사용 가능

**설치 방법:**
```bash
# Go 필요
bash install.sh
# 또는
make build install
```

macOS와 달리 Linux는 추가 네트워크 도구(`docker-mac-net-connect`) 없이 모든 기능이 즉시 동작합니다!

## 도메인 TLD 권장사항

**권장 TLD:**
- `.test` - RFC 2606 테스트용 예약 ✅
- `.localhost` - 로컬 전용 ✅
- `.internal` - 사설 네트워크용 ✅

**피해야 할 TLD:**
- `.local` - macOS mDNS와 충돌 (DNS 조회 느림)
- `.dev` - Google 소유, HTTPS 강제
- `.app` - Google 소유, HTTPS 강제

## 설정 파일

글로벌 설정은 `~/.docker-bootapp/projects.json`에 저장됩니다:

```json
{
  "myproject": {
    "path": "/path/to/project",
    "subnet": "172.18.0.0/16",
    "domain": "myproject.local"
  },
  "another-project": {
    "path": "/path/to/another",
    "subnet": "172.19.0.0/16",
    "domain": "another.local"
  }
}
```

각 프로젝트는 고유한 서브넷(172.18.x.x ~ 172.31.x.x)을 받아 프로젝트 간 IP 충돌을 방지합니다.

## 라이센스

MIT
