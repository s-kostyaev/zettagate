# Maintainer:  <s-kostyaev@ngs>
pkgname=zettagate-git
pkgver=0.2
pkgrel=1
pkgdesc="web server for using zfs from lxc containers"
arch=('i686' 'x86_64')
url="https://github.com/s-kostyaev/zettagate"
license=('unknown')
depends=('git')
makedepends=('go')
backup=('etc/zettagate.toml')
branch='dev'
source=("${pkgname}::git+https://github.com/s-kostyaev/zettagate#branch=${branch}")
md5sums=('SKIP')
noextract=()
build() {
  go get github.com/gin-gonic/gin
  go get github.com/theairkit/runcmd
  go get github.com/BurntSushi/toml
  go get github.com/op/go-logging
  cd ${srcdir}/${pkgname}
  go build -o zettagate
}

package() {
  install -D -m 755 ${srcdir}/${pkgname}/zettagate ${pkgdir}/usr/bin/zettagate
  install -D -m 644 ${srcdir}/${pkgname}/zettagate.toml ${pkgdir}/etc/zettagate.toml
  install -D -m 644 ${srcdir}/${pkgname}/zettagate.service ${pkgdir}/usr/lib/systemd/system/zettagate.service
}
