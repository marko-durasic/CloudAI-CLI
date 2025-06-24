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

# Start Ollama service
sudo systemctl enable ollama
sudo systemctl start ollama

# Wait for Ollama to start
sleep 5

# Pull recommended models
echo "📥 Downloading models..."
ollama pull llama3.2:3b      # Fast, good quality
ollama pull phi3:mini        # Very fast, smaller
ollama pull llama3.2:1b      # Fastest, basic quality

# Configure firewall for remote access
sudo ufw allow 11434/tcp
sudo ufw --force enable

# Get instance IP
INSTANCE_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

echo "✅ Setup complete!"
echo "🌐 Ollama API URL: http://$INSTANCE_IP:11434"
echo "🔧 To use with CloudAI-CLI:"
echo "   export OLLAMA_URL=http://$INSTANCE_IP:11434"
echo "   cloudai setup-interactive"
echo ""
echo "💡 Test connection:"
echo "   curl http://$INSTANCE_IP:11434/api/tags" 