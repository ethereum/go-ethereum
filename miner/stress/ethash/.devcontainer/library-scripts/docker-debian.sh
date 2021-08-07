#!/usr/bin/env bash
#-------------------------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See https://go.microsoft.com/fwlink/?linkid=2090316 for license information.
#-------------------------------------------------------------------------------------------------------------
#
# Docs: https://github.com/microsoft/vscode-dev-containers/blob/main/script-library/docs/docker.md
# Maintainer: The VS Code and Codespaces Teams
#
# Syntax: ./docker-debian.sh [enable non-root docker socket access flag] [source socket] [target socket] [non-root user] [use moby]

ENABLE_NONROOT_DOCKER=${1:-"true"}
SOURCE_SOCKET=${2:-"/var/run/docker-host.sock"}
TARGET_SOCKET=${3:-"/var/run/docker.sock"}
USERNAME=${4:-"automatic"}
USE_MOBY=${5:-"true"}
MICROSOFT_GPG_KEYS_URI="https://packages.microsoft.com/keys/microsoft.asc"

set -e

if [ "$(id -u)" -ne 0 ]; then
    echo -e 'Script must be run as root. Use sudo, su, or add "USER root" to your Dockerfile before running this script.'
    exit 1
fi

# Determine the appropriate non-root user
if [ "${USERNAME}" = "auto" ] || [ "${USERNAME}" = "automatic" ]; then
    USERNAME=""
    POSSIBLE_USERS=("vscode" "node" "codespace" "$(awk -v val=1000 -F ":" '$3==val{print $1}' /etc/passwd)")
    for CURRENT_USER in ${POSSIBLE_USERS[@]}; do
        if id -u ${CURRENT_USER} > /dev/null 2>&1; then
            USERNAME=${CURRENT_USER}
            break
        fi
    done
    if [ "${USERNAME}" = "" ]; then
        USERNAME=root
    fi
elif [ "${USERNAME}" = "none" ] || ! id -u ${USERNAME} > /dev/null 2>&1; then
    USERNAME=root
fi

# Get central common setting
get_common_setting() {
    if [ "${common_settings_file_loaded}" != "true" ]; then
        curl -sfL "https://aka.ms/vscode-dev-containers/script-library/settings.env" 2>/dev/null -o /tmp/vsdc-settings.env || echo "Could not download settings file. Skipping."
        common_settings_file_loaded=true
    fi
    if [ -f "/tmp/vsdc-settings.env" ]; then
        local multi_line=""
        if [ "$2" = "true" ]; then multi_line="-z"; fi
        local result="$(grep ${multi_line} -oP "$1=\"?\K[^\"]+" /tmp/vsdc-settings.env | tr -d '\0')"
        if [ ! -z "${result}" ]; then declare -g $1="${result}"; fi
    fi
    echo "$1=${!1}"
}

# Function to run apt-get if needed
apt_get_update_if_needed()
{
    if [ ! -d "/var/lib/apt/lists" ] || [ "$(ls /var/lib/apt/lists/ | wc -l)" = "0" ]; then
        echo "Running apt-get update..."
        apt-get update
    else
        echo "Skipping apt-get update."
    fi
}

# Checks if packages are installed and installs them if not
check_packages() {
    if ! dpkg -s "$@" > /dev/null 2>&1; then
        apt_get_update_if_needed
        apt-get -y install --no-install-recommends "$@"
    fi
}

# Ensure apt is in non-interactive to avoid prompts
export DEBIAN_FRONTEND=noninteractive

# Install dependencies
check_packages apt-transport-https curl ca-certificates gnupg2

# Install Docker / Moby CLI if not already installed
if type docker > /dev/null 2>&1; then
    echo "Docker / Moby CLI already installed."
else
    # Source /etc/os-release to get OS info
    . /etc/os-release
    if [ "${USE_MOBY}" = "true" ]; then
        # Import key safely (new 'signed-by' method rather than deprecated apt-key approach) and install
        get_common_setting MICROSOFT_GPG_KEYS_URI
        curl -sSL ${MICROSOFT_GPG_KEYS_URI} | gpg --dearmor > /usr/share/keyrings/microsoft-archive-keyring.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/microsoft-archive-keyring.gpg] https://packages.microsoft.com/repos/microsoft-${ID}-${VERSION_CODENAME}-prod ${VERSION_CODENAME} main" > /etc/apt/sources.list.d/microsoft.list
        apt-get update
        apt-get -y install --no-install-recommends moby-cli moby-buildx moby-compose
    else
        # Import key safely (new 'signed-by' method rather than deprecated apt-key approach) and install
        curl -fsSL https://download.docker.com/linux/${ID}/gpg | gpg --dearmor > /usr/share/keyrings/docker-archive-keyring.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/${ID} ${VERSION_CODENAME} stable" > /etc/apt/sources.list.d/docker.list
        apt-get update
        apt-get -y install --no-install-recommends docker-ce-cli
    fi
fi

# Install Docker Compose if not already installed  and is on a supported architecture
if type docker-compose > /dev/null 2>&1; then
    echo "Docker Compose already installed."
else
    TARGET_COMPOSE_ARCH="$(uname -m)"
    if [ "${TARGET_COMPOSE_ARCH}" = "amd64" ]; then
        TARGET_COMPOSE_ARCH="x86_64"
    fi
    if [ "${TARGET_COMPOSE_ARCH}" != "x86_64" ]; then
        # Use pip to get a version that runns on this architecture
        if ! dpkg -s python3-minimal python3-pip libffi-dev python3-venv pipx > /dev/null 2>&1; then
            apt_get_update_if_needed
            apt-get -y install python3-minimal python3-pip libffi-dev python3-venv pipx
        fi
        export PIPX_HOME=/usr/local/pipx
        mkdir -p ${PIPX_HOME}
        export PIPX_BIN_DIR=/usr/local/bin
        export PIP_CACHE_DIR=/tmp/pip-tmp/cache
        pipx install --system-site-packages --pip-args '--no-cache-dir --force-reinstall' docker-compose
        rm -rf /tmp/pip-tmp
    else 
        LATEST_COMPOSE_VERSION=$(basename "$(curl -fsSL -o /dev/null -w "%{url_effective}" https://github.com/docker/compose/releases/latest)")
        curl -fsSL "https://github.com/docker/compose/releases/download/${LATEST_COMPOSE_VERSION}/docker-compose-$(uname -s)-${TARGET_COMPOSE_ARCH}" -o /usr/local/bin/docker-compose
        chmod +x /usr/local/bin/docker-compose
    fi
fi

# If init file already exists, exit
if [ -f "/usr/local/share/docker-init.sh" ]; then
    exit 0
fi

# By default, make the source and target sockets the same
if [ "${SOURCE_SOCKET}" != "${TARGET_SOCKET}" ]; then
    touch "${SOURCE_SOCKET}"
    ln -s "${SOURCE_SOCKET}" "${TARGET_SOCKET}"
fi

# Add a stub if not adding non-root user access, user is root
if [ "${ENABLE_NONROOT_DOCKER}" = "false" ] || [ "${USERNAME}" = "root" ]; then
    echo '/usr/bin/env bash -c "\$@"' > /usr/local/share/docker-init.sh
    chmod +x /usr/local/share/docker-init.sh
    exit 0
fi

# If enabling non-root access and specified user is found, setup socat and add script
chown -h "${USERNAME}":root "${TARGET_SOCKET}"        
if ! dpkg -s socat > /dev/null 2>&1; then
    apt_get_update_if_needed
    apt-get -y install socat
fi
tee /usr/local/share/docker-init.sh > /dev/null \
<< EOF 
#!/usr/bin/env bash
#-------------------------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See https://go.microsoft.com/fwlink/?linkid=2090316 for license information.
#-------------------------------------------------------------------------------------------------------------

set -e

SOCAT_PATH_BASE=/tmp/vscr-docker-from-docker
SOCAT_LOG=\${SOCAT_PATH_BASE}.log
SOCAT_PID=\${SOCAT_PATH_BASE}.pid

# Wrapper function to only use sudo if not already root
sudoIf()
{
    if [ "\$(id -u)" -ne 0 ]; then
        sudo "\$@"
    else
        "\$@"
    fi
}

# Log messages
log()
{
    echo -e "[\$(date)] \$@" | sudoIf tee -a \${SOCAT_LOG} > /dev/null
}

echo -e "\n** \$(date) **" | sudoIf tee -a \${SOCAT_LOG} > /dev/null
log "Ensuring ${USERNAME} has access to ${SOURCE_SOCKET} via ${TARGET_SOCKET}"

# If enabled, try to add a docker group with the right GID. If the group is root, 
# fall back on using socat to forward the docker socket to another unix socket so 
# that we can set permissions on it without affecting the host.
if [ "${ENABLE_NONROOT_DOCKER}" = "true" ] && [ "${SOURCE_SOCKET}" != "${TARGET_SOCKET}" ] && [ "${USERNAME}" != "root" ] && [ "${USERNAME}" != "0" ]; then
    SOCKET_GID=\$(stat -c '%g' ${SOURCE_SOCKET})
    if [ "\${SOCKET_GID}" != "0" ]; then
        log "Adding user to group with GID \${SOCKET_GID}."
        if [ "\$(cat /etc/group | grep :\${SOCKET_GID}:)" = "" ]; then
            sudoIf groupadd --gid \${SOCKET_GID} docker-host
        fi
        # Add user to group if not already in it
        if [ "\$(id ${USERNAME} | grep -E "groups.*(=|,)\${SOCKET_GID}\(")" = "" ]; then
            sudoIf usermod -aG \${SOCKET_GID} ${USERNAME}
        fi
    else
        # Enable proxy if not already running
        if [ ! -f "\${SOCAT_PID}" ] || ! ps -p \$(cat \${SOCAT_PID}) > /dev/null; then
            log "Enabling socket proxy."
            log "Proxying ${SOURCE_SOCKET} to ${TARGET_SOCKET} for vscode"
            sudoIf rm -rf ${TARGET_SOCKET}
            (sudoIf socat UNIX-LISTEN:${TARGET_SOCKET},fork,mode=660,user=${USERNAME} UNIX-CONNECT:${SOURCE_SOCKET} 2>&1 | sudoIf tee -a \${SOCAT_LOG} > /dev/null & echo "\$!" | sudoIf tee \${SOCAT_PID} > /dev/null)
        else
            log "Socket proxy already running."
        fi
    fi
    log "Success"
fi

# Execute whatever commands were passed in (if any). This allows us 
# to set this script to ENTRYPOINT while still executing the default CMD.
set +e
exec "\$@"
EOF
chmod +x /usr/local/share/docker-init.sh
chown ${USERNAME}:root /usr/local/share/docker-init.sh
echo "Done!"