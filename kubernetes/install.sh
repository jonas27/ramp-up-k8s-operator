#!/bin/bash
## mostly from https://blog.kubesimplify.com/how-to-install-a-kubernetes-cluster-with-kubeadm-containerd-and-cilium-a-hands-on-guide

apt-get update && apt-get upgrade -y

## kubernetes
apt-get install -y apt-transport-https ca-certificates curl &&
  curl -fsSL https://dl.k8s.io/apt/doc/apt-key.gpg | gpg --dearmor -o /etc/apt/keyrings/kubernetes-archive-keyring.gpg &&
  echo "deb [signed-by=/etc/apt/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list

apt-get update &&
  apt-get install -y kubelet kubeadm kubectl &&
  apt-mark hold kubelet kubeadm kubectl

## network stuff
modprobe br_netfilter

cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sysctl --system

## Install containerd
apt install curl gnupg2 software-properties-common apt-transport-https ca-certificates -y
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
apt update && apt install containerd.io -y

## Config containerd
mkdir -p /etc/containerd
containerd config default >/etc/containerd/config.toml
systemctl restart containerd
systemctl enable containerd

## turn swap off
swapoff -a &&
  sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab # see https://brandonwillmott.com/2020/10/15/permanently-disable-swap-for-kubernetes-cluster/

## Init cluster
kubeadm init --pod-network-cidr=10.1.1.0/24 --apiserver-advertise-address $IP

## Join cluster `
# kubeadm join $IP:6443 --token $TOKEN --discovery-token-ca-cert-hash $CA
