#!/bin/sh
LATEST_VERSION="0.0.0-alpha"

function install_osx() {
  brew tap outlaw/homebrew-tap
  brew install outlaw/homebrew-tap/roo
}

function install_linux() {
  curl -sL "https://github.com/outlaw/roo/releases/download/$LATEST_VERSION/roo-linux-x86_64.tar.gz" | tar xzf -
  chmod +x roo
  mv roo /usr/local/bin
}

function install() {
  echo "*** Installing roo $LATEST_VERSION"
  if [ "$(uname -s)" == "Darwin" ]; then
    install_osx
  else
    install_linux
  fi

  echo "*** Installed roo $LATEST_VERSION"
}

install
