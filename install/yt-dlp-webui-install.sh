#!/usr/bin/env bash

# Copyright (c) 2021-2026 community-scripts ORG
# Author: CrazyWolf13
# License: MIT | https://github.com/community-scripts/ProxmoxVE/raw/main/LICENSE
# Source: https://github.com/NStevenU/yt-dlp-web-ui

source /dev/stdin <<<"$FUNCTIONS_FILE_PATH"
color
verb_ip6
catch_errors
setting_up_container
network_check
update_os

msg_info "Installing Dependencies"
$STD apt install -y ffmpeg nodejs
msg_ok "Installed Dependencies"

fetch_and_deploy_gh_release "yt-dlp-webui" "NStevenU/yt-dlp-web-ui" "singlefile" "latest" "/usr/local/bin" "yt-dlp-webui_linux-$(arch_resolve)"
fetch_and_deploy_gh_release "yt-dlp" "yt-dlp/yt-dlp" "singlefile" "latest" "/usr/local/bin" "yt-dlp"

msg_info "Setting up YT-DLP-WEBUI"
mkdir -p /opt/yt-dlp-webui
mkdir -p /downloads
RPC_PASSWORD=$(random_password 21)
cat <<EOF >~/yt-dlp-webui.creds
yt-dlp-webui-Credentials
Username: admin
Password: ${RPC_PASSWORD}
EOF

cat <<EOF >/opt/yt-dlp-webui/config.conf
server:
  host: 0.0.0.0
  port: 3033
  queue_size: 4

paths:
  download_path: /downloads
  downloader_path: /usr/local/bin/yt-dlp
  local_database_path: /opt/yt-dlp-webui

authentication:
  require_auth: true
  username: admin
  password_hash: "${RPC_PASSWORD}"

logging:
  enable_file_logging: false
  log_path: /opt/yt-dlp-webui/yt-dlp-webui.log
EOF
msg_ok "Set up YT-DLP-WEBUI"

msg_info "Creating Service"
JWT_SECRET=$(random_password 32)
cat <<EOF >/etc/systemd/system/yt-dlp-webui.service
[Unit]
Description=yt-dlp-webui service file
After=network.target

[Service]
WorkingDirectory=/opt/yt-dlp-webui
Environment=JWT_SECRET=${JWT_SECRET}
ExecStart=/usr/local/bin/yt-dlp-webui --conf /opt/yt-dlp-webui/config.conf
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
systemctl enable -q --now yt-dlp-webui
msg_ok "Created Service"

motd_ssh
customize
cleanup_lxc
