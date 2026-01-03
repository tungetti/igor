#!/bin/bash
# =============================================================================
# NVIDIA Driver & CUDA Installation Script for Ubuntu 24.04 LTS
# =============================================================================
#
# This script automates the installation of NVIDIA drivers and CUDA toolkit
# on a fresh Ubuntu 24.04 LTS installation.
#
# Based on: https://github.com/oddmario/NVIDIA-Ubuntu-Driver-Guide
#
# Features:
# - Checks for latest driver versions from graphics-drivers PPA
# - Supports both production and new feature branch drivers
# - Installs CUDA toolkit for ML/AI workloads
# - Configures Wayland support properly
# - Includes verification and rollback options
#
# Usage:
#   ./install_nvidia.sh              # Interactive mode
#   ./install_nvidia.sh --auto       # Auto-install recommended driver
#   ./install_nvidia.sh --check      # Check available versions only
#   ./install_nvidia.sh --uninstall  # Uninstall NVIDIA drivers
#
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# Configuration
# =============================================================================

# PPA Launchpad page for checking versions
PPA_URL="https://launchpad.net/~graphics-drivers/+archive/ubuntu/ppa"

# Default driver version (production branch) - updated Dec 2025
DEFAULT_DRIVER_VERSION="570"

# Minimum recommended driver for Wayland explicit sync support
MIN_WAYLAND_DRIVER="555"

# CUDA version to install (leave empty to skip CUDA installation)
CUDA_VERSION="12-6"

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

check_ubuntu() {
    if [ ! -f /etc/os-release ]; then
        log_error "Cannot detect OS. This script is for Ubuntu only."
        exit 1
    fi
    
    . /etc/os-release
    
    if [ "$ID" != "ubuntu" ]; then
        log_error "This script is designed for Ubuntu. Detected: $ID"
        exit 1
    fi
    
    log_info "Detected: $PRETTY_NAME"
    
    # Check for Ubuntu 20.04+
    VERSION_MAJOR=$(echo "$VERSION_ID" | cut -d. -f1)
    if [ "$VERSION_MAJOR" -lt 20 ]; then
        log_error "Ubuntu 20.04 or newer required. Detected: $VERSION_ID"
        exit 1
    fi
}

check_nvidia_gpu() {
    log_info "Checking for NVIDIA GPU..."
    
    if ! lspci | grep -i nvidia > /dev/null 2>&1; then
        log_error "No NVIDIA GPU detected. Please check your hardware."
        exit 1
    fi
    
    GPU_INFO=$(lspci | grep -i nvidia | head -1)
    log_success "Found GPU: $GPU_INFO"
}

get_current_driver() {
    if command -v nvidia-smi &> /dev/null; then
        CURRENT_DRIVER=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null || echo "")
        if [ -n "$CURRENT_DRIVER" ]; then
            echo "$CURRENT_DRIVER"
            return 0
        fi
    fi
    echo "none"
}

check_available_versions() {
    log_info "Checking available driver versions from PPA..."
    
    echo ""
    echo "============================================================"
    echo "Available NVIDIA Driver Versions (graphics-drivers PPA)"
    echo "============================================================"
    echo ""
    echo "Current releases (as of PPA):"
    echo "  - Production branch:    570.x (recommended for stability)"
    echo "  - New feature branch:   575.x (latest features)"
    echo "  - Beta branch:          580.x (experimental)"
    echo ""
    echo "Legacy releases:"
    echo "  - 550.x - Previous production"
    echo "  - 470.x - Kepler GPUs (GTX 600/700 series)"
    echo "  - 390.x - Fermi GPUs (GTX 400/500 series)"
    echo ""
    echo "Check latest versions at:"
    echo "  $PPA_URL"
    echo ""
    echo "============================================================"
    
    # Try to get actual available versions from apt if PPA is added
    if apt-cache search nvidia-driver 2>/dev/null | grep -q nvidia-driver; then
        echo ""
        log_info "Drivers available in your current repositories:"
        apt-cache search nvidia-driver | grep "^nvidia-driver-[0-9]" | sort -t'-' -k3 -n | tail -10
    fi
}

# =============================================================================
# Installation Functions
# =============================================================================

install_dependencies() {
    log_info "Installing required dependencies..."
    
    apt-get update
    apt-get install -y \
        pkg-config \
        libglvnd-dev \
        dkms \
        build-essential \
        libegl-dev \
        libegl1 \
        libgl-dev \
        libgl1 \
        libgles-dev \
        libgles1 \
        libglvnd-core-dev \
        libglx-dev \
        libopengl-dev \
        gcc \
        make \
        linux-headers-$(uname -r) \
        software-properties-common \
        curl \
        wget
    
    log_success "Dependencies installed successfully"
}

remove_existing_drivers() {
    log_info "Removing any existing NVIDIA drivers..."
    
    # Remove drivers installed via APT
    apt-get remove --purge -y '^nvidia-.*' 2>/dev/null || true
    apt-get remove --purge -y '^libnvidia-.*' 2>/dev/null || true
    apt-get autoremove -y
    
    # Ensure ubuntu-drivers-common is installed
    apt-get install -y ubuntu-drivers-common
    
    log_success "Existing drivers removed"
}

add_graphics_ppa() {
    log_info "Adding graphics-drivers PPA repository..."
    
    # Check if PPA already exists
    if ! grep -q "graphics-drivers/ppa" /etc/apt/sources.list.d/*.list 2>/dev/null; then
        add-apt-repository -y ppa:graphics-drivers/ppa
    else
        log_info "PPA already added"
    fi
    
    apt-get update
    log_success "PPA repository added and updated"
}

install_nvidia_driver() {
    local DRIVER_VERSION=${1:-$DEFAULT_DRIVER_VERSION}
    
    log_info "Installing NVIDIA driver version $DRIVER_VERSION..."
    
    # Install the driver
    apt-get install -y "nvidia-driver-${DRIVER_VERSION}"
    
    log_success "NVIDIA driver $DRIVER_VERSION installed"
}

install_cuda_toolkit() {
    if [ -z "$CUDA_VERSION" ]; then
        log_info "Skipping CUDA toolkit installation (not configured)"
        return
    fi
    
    log_info "Installing CUDA toolkit $CUDA_VERSION..."
    
    # Add NVIDIA CUDA repository
    # Get Ubuntu version codename
    . /etc/os-release
    UBUNTU_CODENAME=$VERSION_CODENAME
    
    # Download and install CUDA keyring
    CUDA_KEYRING_URL="https://developer.download.nvidia.com/compute/cuda/repos/ubuntu${VERSION_ID//./}/x86_64/cuda-keyring_1.1-1_all.deb"
    
    if ! wget -q "$CUDA_KEYRING_URL" -O /tmp/cuda-keyring.deb 2>/dev/null; then
        log_warning "Could not download CUDA keyring. Trying alternative method..."
        # Alternative: use ubuntu2404 explicitly
        wget -q "https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb" -O /tmp/cuda-keyring.deb
    fi
    
    dpkg -i /tmp/cuda-keyring.deb
    apt-get update
    
    # Install CUDA toolkit
    apt-get install -y "cuda-toolkit-${CUDA_VERSION}"
    
    # Add CUDA to PATH in profile
    if ! grep -q "cuda" /etc/profile.d/cuda.sh 2>/dev/null; then
        cat > /etc/profile.d/cuda.sh << 'EOF'
# CUDA Toolkit paths
export PATH=/usr/local/cuda/bin${PATH:+:${PATH}}
export LD_LIBRARY_PATH=/usr/local/cuda/lib64${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}
EOF
    fi
    
    log_success "CUDA toolkit installed"
    log_info "Please log out and log back in for CUDA PATH to take effect"
}

configure_wayland() {
    log_info "Configuring Wayland support..."
    
    # Enable Wayland in GDM
    if [ -f /etc/gdm3/custom.conf ]; then
        sed -i 's/#WaylandEnable=.*/WaylandEnable=true/' /etc/gdm3/custom.conf
        sed -i 's/WaylandEnable=false/WaylandEnable=true/' /etc/gdm3/custom.conf
    fi
    
    # Create udev rule to enable Wayland with NVIDIA
    ln -sf /dev/null /etc/udev/rules.d/61-gdm.rules 2>/dev/null || true
    
    # Configure GRUB for proper NVIDIA DRM
    GRUB_FILE="/etc/default/grub"
    if [ -f "$GRUB_FILE" ]; then
        # Backup original
        cp "$GRUB_FILE" "${GRUB_FILE}.backup"
        
        # Check if nvidia-drm.modeset is already set
        if ! grep -q "nvidia-drm.modeset=1" "$GRUB_FILE"; then
            # Add NVIDIA kernel parameters
            CURRENT_CMDLINE=$(grep "^GRUB_CMDLINE_LINUX=" "$GRUB_FILE" | sed 's/GRUB_CMDLINE_LINUX="//' | sed 's/"$//')
            NEW_PARAMS="nvidia-drm.modeset=1 nvidia-drm.fbdev=1"
            
            if [ -z "$CURRENT_CMDLINE" ]; then
                sed -i "s/^GRUB_CMDLINE_LINUX=\"\"/GRUB_CMDLINE_LINUX=\"$NEW_PARAMS\"/" "$GRUB_FILE"
            else
                sed -i "s/^GRUB_CMDLINE_LINUX=\".*\"/GRUB_CMDLINE_LINUX=\"$CURRENT_CMDLINE $NEW_PARAMS\"/" "$GRUB_FILE"
            fi
            
            log_info "Updated GRUB configuration"
        fi
        
        # Update GRUB
        update-grub
    fi
    
    # Update initramfs
    update-initramfs -u
    
    log_success "Wayland configuration complete"
}

verify_installation() {
    log_info "Verifying installation..."
    
    echo ""
    echo "============================================================"
    echo "Installation Verification"
    echo "============================================================"
    
    # Check if nvidia-smi works
    if command -v nvidia-smi &> /dev/null; then
        echo ""
        nvidia-smi
        echo ""
        log_success "nvidia-smi is working"
    else
        log_warning "nvidia-smi not found. A reboot may be required."
    fi
    
    # Check CUDA
    if command -v nvcc &> /dev/null; then
        CUDA_VER=$(nvcc --version | grep "release" | awk '{print $6}')
        log_success "CUDA compiler version: $CUDA_VER"
    else
        log_info "CUDA compiler not in PATH (may need to log out/in)"
    fi
    
    echo ""
    echo "============================================================"
}

# =============================================================================
# Uninstallation Function
# =============================================================================

uninstall_nvidia() {
    log_warning "This will remove all NVIDIA drivers from your system."
    read -p "Are you sure you want to continue? (y/N): " confirm
    
    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        log_info "Uninstallation cancelled"
        exit 0
    fi
    
    log_info "Uninstalling NVIDIA drivers..."
    
    # Remove APT packages
    apt-get remove --purge -y '^nvidia-.*' 2>/dev/null || true
    apt-get remove --purge -y '^libnvidia-.*' 2>/dev/null || true
    apt-get remove --purge -y '^cuda-.*' 2>/dev/null || true
    apt-get autoremove -y
    
    # Reinstall ubuntu-drivers-common
    apt-get install -y ubuntu-drivers-common
    
    # Remove GRUB parameters
    GRUB_FILE="/etc/default/grub"
    if [ -f "$GRUB_FILE" ]; then
        sed -i 's/nvidia-drm.modeset=1//g' "$GRUB_FILE"
        sed -i 's/nvidia-drm.fbdev=1//g' "$GRUB_FILE"
        sed -i 's/nvidia.NVreg_EnableGpuFirmware=0//g' "$GRUB_FILE"
        sed -i 's/nvidia.NVreg_PreserveVideoMemoryAllocations=1//g' "$GRUB_FILE"
        # Clean up double spaces
        sed -i 's/  */ /g' "$GRUB_FILE"
        update-grub
    fi
    
    # Update initramfs
    update-initramfs -u
    
    log_success "NVIDIA drivers uninstalled"
    log_warning "Please reboot your system to complete the uninstallation"
}

# =============================================================================
# Main Script
# =============================================================================

print_banner() {
    echo ""
    echo "============================================================"
    echo "  NVIDIA Driver & CUDA Installation Script"
    echo "  For Ubuntu 24.04 LTS"
    echo "============================================================"
    echo ""
}

print_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --auto          Auto-install recommended driver ($DEFAULT_DRIVER_VERSION)"
    echo "  --version VER   Install specific driver version (e.g., 570, 575)"
    echo "  --check         Check available versions only"
    echo "  --uninstall     Uninstall NVIDIA drivers"
    echo "  --no-cuda       Skip CUDA toolkit installation"
    echo "  --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  sudo $0                    # Interactive installation"
    echo "  sudo $0 --auto             # Auto-install driver $DEFAULT_DRIVER_VERSION"
    echo "  sudo $0 --version 575      # Install driver 575"
    echo "  sudo $0 --check            # Check versions without installing"
    echo ""
}

main() {
    print_banner
    
    # Parse arguments
    AUTO_MODE=false
    CHECK_ONLY=false
    UNINSTALL=false
    DRIVER_VERSION="$DEFAULT_DRIVER_VERSION"
    SKIP_CUDA=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --auto)
                AUTO_MODE=true
                shift
                ;;
            --version)
                DRIVER_VERSION="$2"
                shift 2
                ;;
            --check)
                CHECK_ONLY=true
                shift
                ;;
            --uninstall)
                UNINSTALL=true
                shift
                ;;
            --no-cuda)
                SKIP_CUDA=true
                CUDA_VERSION=""
                shift
                ;;
            --help|-h)
                print_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Check only mode
    if [ "$CHECK_ONLY" = true ]; then
        check_ubuntu
        check_nvidia_gpu
        CURRENT=$(get_current_driver)
        if [ "$CURRENT" != "none" ]; then
            log_info "Currently installed driver: $CURRENT"
        else
            log_info "No NVIDIA driver currently installed"
        fi
        check_available_versions
        exit 0
    fi
    
    # Uninstall mode
    if [ "$UNINSTALL" = true ]; then
        check_root
        uninstall_nvidia
        exit 0
    fi
    
    # Installation mode
    check_root
    check_ubuntu
    check_nvidia_gpu
    
    # Show current driver
    CURRENT=$(get_current_driver)
    if [ "$CURRENT" != "none" ]; then
        log_info "Currently installed driver: $CURRENT"
    fi
    
    # Interactive mode
    if [ "$AUTO_MODE" = false ]; then
        check_available_versions
        echo ""
        read -p "Enter driver version to install (default: $DEFAULT_DRIVER_VERSION): " USER_VERSION
        if [ -n "$USER_VERSION" ]; then
            DRIVER_VERSION="$USER_VERSION"
        fi
        
        if [ "$SKIP_CUDA" = false ]; then
            read -p "Install CUDA toolkit? (Y/n): " INSTALL_CUDA
            if [ "$INSTALL_CUDA" = "n" ] || [ "$INSTALL_CUDA" = "N" ]; then
                CUDA_VERSION=""
            fi
        fi
    fi
    
    echo ""
    echo "============================================================"
    echo "Installation Summary"
    echo "============================================================"
    echo "  Driver version: $DRIVER_VERSION"
    echo "  CUDA toolkit:   ${CUDA_VERSION:-skipped}"
    echo "============================================================"
    echo ""
    
    if [ "$AUTO_MODE" = false ]; then
        read -p "Proceed with installation? (Y/n): " CONFIRM
        if [ "$CONFIRM" = "n" ] || [ "$CONFIRM" = "N" ]; then
            log_info "Installation cancelled"
            exit 0
        fi
    fi
    
    # Start installation
    log_info "Starting installation..."
    echo ""
    
    install_dependencies
    remove_existing_drivers
    add_graphics_ppa
    install_nvidia_driver "$DRIVER_VERSION"
    
    if [ -n "$CUDA_VERSION" ]; then
        install_cuda_toolkit
    fi
    
    configure_wayland
    
    echo ""
    echo "============================================================"
    echo "  Installation Complete!"
    echo "============================================================"
    echo ""
    log_success "NVIDIA driver $DRIVER_VERSION has been installed"
    
    if [ -n "$CUDA_VERSION" ]; then
        log_success "CUDA toolkit $CUDA_VERSION has been installed"
    fi
    
    echo ""
    log_warning "IMPORTANT: You must reboot your system for changes to take effect!"
    echo ""
    
    read -p "Reboot now? (y/N): " REBOOT_NOW
    if [ "$REBOOT_NOW" = "y" ] || [ "$REBOOT_NOW" = "Y" ]; then
        log_info "Rebooting..."
        reboot
    else
        log_info "Please reboot manually when ready: sudo reboot"
    fi
}

# Run main function
main "$@"
