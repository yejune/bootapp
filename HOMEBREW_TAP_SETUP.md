# Homebrew Tap 설정 가이드

이 가이드는 `brew install yejune/tap/docker-bootapp` 형태로 설치할 수 있게 Homebrew Tap을 설정하는 방법입니다.

## 1단계: homebrew-tap 저장소 생성

GitHub에 새 저장소를 만듭니다:
- 저장소 이름: `homebrew-tap` (반드시 `homebrew-` 접두사 필요)
- Public 저장소로 생성

```bash
# 로컬에서 저장소 생성
mkdir homebrew-tap
cd homebrew-tap
git init
```

## 2단계: Formula 파일 복사

현재 프로젝트의 `docker-bootapp.rb` 파일을 homebrew-tap 저장소에 복사:

```bash
cp /path/to/docker-bootapp/docker-bootapp.rb .
git add docker-bootapp.rb
git commit -m "Add docker-bootapp formula"
```

## 3단계: GitHub에 푸시

```bash
git remote add origin https://github.com/yejune/homebrew-tap.git
git branch -M main
git push -u origin main
```

## 4단계: 첫 릴리스 만들기

docker-bootapp 프로젝트에서 릴리스를 만듭니다:

```bash
cd /path/to/docker-bootapp
git tag v1.0.0
git push origin v1.0.0
```

또는 GitHub UI에서:
1. Releases 탭 이동
2. "Create a new release" 클릭
3. Tag: `v1.0.0`
4. Title: `v1.0.0`
5. "Publish release" 클릭

## 5단계: SHA256 해시 업데이트

릴리스가 생성되면 tarball의 SHA256 해시를 계산:

```bash
# tarball 다운로드
curl -L -o docker-bootapp-1.0.0.tar.gz \
  https://github.com/yejune/docker-bootapp/archive/refs/tags/v1.0.0.tar.gz

# SHA256 계산
shasum -a 256 docker-bootapp-1.0.0.tar.gz
```

나온 해시값을 `docker-bootapp.rb` 파일의 `sha256` 줄에 넣습니다:

```ruby
sha256 "여기에_해시값_붙여넣기"
```

그리고 커밋:

```bash
cd homebrew-tap
git add docker-bootapp.rb
git commit -m "Update SHA256 for v1.0.0"
git push
```

## 6단계: 설치 테스트

이제 사용자들이 다음 명령어로 설치할 수 있습니다:

```bash
# Tap 추가 (처음 한 번만)
brew tap yejune/tap

# 설치
brew install docker-bootapp
```

또는 한 줄로:

```bash
brew install yejune/tap/docker-bootapp
```

## 자동화: GitHub Actions로 SHA256 자동 업데이트

새 릴리스가 나올 때마다 자동으로 Formula를 업데이트하려면, `.github/workflows/update-formula.yml` 파일을 참고하세요.

## 업데이트 배포

새 버전을 릴리스할 때:

1. docker-bootapp 저장소에서 새 태그 생성:
   ```bash
   git tag v1.0.1
   git push origin v1.0.1
   ```

2. homebrew-tap 저장소의 Formula 업데이트:
   - `url` 줄의 버전 업데이트
   - 새 SHA256 계산 및 업데이트

3. 커밋 & 푸시

사용자들은 다음 명령어로 업데이트:
```bash
brew update
brew upgrade docker-bootapp
```

## 참고

- Homebrew Formula 문서: https://docs.brew.sh/Formula-Cookbook
- Tap 생성 가이드: https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap
