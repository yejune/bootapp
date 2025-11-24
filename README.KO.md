# docker-bootapp

멀티 프로젝트 Docker 네트워킹을 쉽게 관리하는 Docker CLI 플러그인.

자동으로 관리:
- 프로젝트별 고유 서브넷 할당 (충돌 방지)
- DOMAIN 설정이 있는 컨테이너의 /etc/hosts 항목
- macOS 스마트 라우트 설정 (연결 테스트 후 라우트 추가)

## 설치

### 빠른 설치
```bash
make build install
```

### 수동 설치
```bash
make build
cp build/docker-bootapp ~/.docker/cli-plugins/docker-bootapp
chmod +x ~/.docker/cli-plugins/docker-bootapp
```

## 사용법

### 컨테이너 시작
```bash
# docker-compose.yml 자동 감지
docker bootapp up

# compose 파일 지정
docker bootapp -f docker-compose.local.yml up
```

여러 compose 파일이 있으면 선택 프롬프트가 표시됩니다:
```
Multiple compose files found:
  [1] docker-compose.yml
  [2] docker-compose.local.yml
  [3] docker-compose.prod.yml

Select file (1-3):
```

지원하는 파일 패턴:
- `docker-compose.yml`, `docker-compose.yaml`
- `docker-compose.*.yml`, `docker-compose.*.yaml` (예: docker-compose.local.yml)
- `compose.yml`, `compose.yaml`

실행 시:
1. 프로젝트에 고유 서브넷 할당 (172.18-31.x.x 범위)
2. docker-compose 파일에서 DOMAIN/SSL_DOMAINS 설정 파싱
3. docker-compose up으로 컨테이너 시작
4. 기본 compose 네트워크에서 컨테이너 IP 감지
5. 도메인 설정이 있는 컨테이너를 /etc/hosts에 추가
6. 필요 시 라우팅 설정 (macOS)

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

## macOS 네트워킹

macOS에서 컨테이너 IP에 직접 접근하려면:

### 방법 1: docker-mac-net-connect (권장)
```bash
brew install chipmk/tap/docker-mac-net-connect
sudo brew services start docker-mac-net-connect
```

### 방법 2: 자동 라우트
작동하는 라우트가 없으면 bootapp이 자동으로:
1. 라우트 존재 여부 확인 및 연결 테스트
2. 작동하지 않으면 Docker VM 게이트웨이를 통한 라우트 추가
3. 이미 연결이 되면 건너뜀

## Linux

추가 설정 불필요 - Docker 네트워킹이 기본적으로 작동합니다.

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
