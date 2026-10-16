#!/bin/bash
# EC2 Ollama Setup Script
# Run this on a fresh Ubuntu 22.04 GPU instance (g4dn.xlarge or similar)

set -e

echo "🚀 Setting up Ollama on EC2 GPU instance..."

# Update system
sudo apt update && sudo apt upgrade -y

# Install NVIDIA drivers
sudo apt install -y ubuntu-drivers-common
sudo ubuntu-drivers autoinstall

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
sudo usermod -aG docker $USER

# Install NVIDIA Container Toolkit
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt update && sudo apt install -y nvidia-docker2
sudo systemctl restart docker

# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Bind Ollama to localhost; remote users must connect over an SSH tunnel.
sudo mkdir -p /etc/systemd/system/ollama.service.d
printf '[Service]\nEnvironment="OLLAMA_HOST=127.0.0.1:11434"\n' | sudo tee /etc/systemd/system/ollama.service.d/override.conf >/dev/null
sudo systemctl daemon-reload
sudo systemctl enable ollama
sudo systemctl restart ollama

# Wait for Ollama to start
sleep 5

# Pull recommended models
echo "📥 Downloading models..."
ollama pull llama3.2:3b      # Fast, good quality
ollama pull phi3:mini        # Very fast, smaller
ollama pull llama3.2:1b      # Fastest, basic quality

# Firewall: allow SSH, do not expose the unauthenticated Ollama API.
sudo ufw allow 22/tcp
sudo ufw --force enable

# Get instance IP
INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

echo "✅ Setup complete!"
echo "🔐 From your trusted workstation, open an SSH tunnel:"
echo "   ssh -N -L 11434:127.0.0.1:11434 ubuntu@$INSTANCE_IP"
echo "🔧 In another terminal on that workstation:"
echo "   export OLLAMA_URL=http://127.0.0.1:11434"
echo "   cloudai setup-interactive"
echo ""
echo "💡 Test connection:"
echo "   curl http://127.0.0.1:11434/api/tags" 