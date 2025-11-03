#!/bin/bash
set -x

curl -L -o nvim.tar.gz https://github.com/neovim/neovim/releases/download/v0.11.4/nvim-linux-arm64.tar.gz
tar xzvf nvim.tar.gz
cp -r nvim-linux-arm64/* /usr/.

apt update
apt install -y nodejs
git clone https://github.com/LazyVim/starter ~/.config/nvim
rm -rf ~/.config/nvim/.git

apt install -y libevent-core-2.1-7
curl -L -o ~/.tmux.conf https://raw.githubusercontent.com/gpakosz/.tmux/refs/heads/master/.tmux.conf
curl -L -o ~/.tmux.conf.local https://raw.githubusercontent.com/gpakosz/.tmux/refs/heads/master/.tmux.conf.local

curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/linux/arm64
chmod +x kubebuilder
mv kubebuilder /usr/local/bin/

KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/linux/arm64/kubectl"
chmod +x kubectl
mv kubectl /usr/local/bin/kubectl

kubebuilder version
go version
kubectl version --client
