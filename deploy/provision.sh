#!/usr/bin/env bash
# Provisioning of the server (ADR 0015; plan recette, lot B), from the deploy/ directory of
# an archive: system, SSH, firewall, updates, mail relay, users, directories, systemd units
# and Caddy, for the environments given in argument. Idempotent: it can be run again at any
# time, and only restarts what it changed.
#
#   sudo ./deploy/provision.sh production recette
#
# The values of the instance come from /etc/tribe-menus/instance.conf (plan recette, D4),
# read as data, never evaluated by the shell; the files made from them are written here,
# never by hand. Secrets are not here: tribe-menus-credential reads them at the keyboard
# (D5).
#
# --render <directory> --instance <file> <environment>... writes the files made from an
# instance file into the directory, without root and without touching the system: for the
# tests and for review.
#
# Procedures, and why each setting: deploy/README.md.
set -euo pipefail
# put, last of a pipeline, runs in this shell: it sets changed.
shopt -s lastpipe

readonly INSTANCE_CONF=/etc/tribe-menus/instance.conf
DEPLOY_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
readonly DEPLOY_DIR
readonly BIN_DIR=/usr/local/sbin
readonly SHARE_DIR=/usr/share/tribe-menus
readonly STATE_DIR=/var/lib/tribe-menus/provision
# Marks that root may no longer connect by SSH (ADR 0025, point 3).
readonly ROOT_SSH_CLOSED=$STATE_DIR/root-ssh-closed
readonly SCRIPTS=(tribe-menus-credential tribe-menus-admin)
readonly DEBIAN_PACKAGES=(sudo tzdata nftables unattended-upgrades msmtp-mta sqlite3 jq curl unzip ca-certificates)

die() {
	printf 'provision: %s\n' "$*" >&2
	exit 1
}

step() {
	printf '\n==> %s\n' "$*"
}

usage() {
	cat >&2 <<'EOF'
Usage: provision.sh <environment>...
       provision.sh --render <directory> --instance <file> <environment>...
Environments: production, recette.
EOF
	exit 2
}

# --- Values of the instance ------------------------------------------------------------

# Keys of instance.conf, all required, and the pattern of their values. A value matching
# its pattern holds no quote, backslash, space or $: it can be written as is into JSON, the
# Caddyfile and the msmtp configuration.
readonly LABEL='[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?'
readonly HOST="$LABEL(\\.$LABEL)+"
readonly EMAIL="[a-z0-9]([a-z0-9._+-]{0,62}[a-z0-9])?@$HOST"
declare -rA PATTERNS=(
	[ADMIN_USER]='^[a-z][a-z0-9-]{0,30}$'
	[ADMIN_EMAIL]="^$EMAIL\$"
	[PRODUCTION_HOST]="^$HOST\$"
	[RECETTE_HOST]="^$HOST\$"
	[SMTP_HOST]="^$HOST\$"
	[SMTP_PORT]='^[1-9][0-9]{0,4}$'
	[APP_SMTP_USER]="^$EMAIL\$"
	[APP_SMTP_FROM]="^$EMAIL\$"
	[SERVER_SMTP_USER]="^$EMAIL\$"
	[SERVER_SMTP_FROM]="^$EMAIL\$"
)

declare -A conf=()

# read_instance reads an instance file into conf. Errors name the line and the key, never
# the value.
read_instance() {
	local file=$1 line key value n=0
	[[ -f $file && -r $file ]] || die "$file: missing or unreadable (template: deploy/instance.example.conf)"
	while IFS= read -r line || [[ -n $line ]]; do
		n=$((n + 1))
		[[ $line =~ ^[[:space:]]*(#.*)?$ ]] && continue
		[[ $line =~ ^([A-Z_]+)=(.*)$ ]] || die "$file, line $n: want KEY=value"
		key=${BASH_REMATCH[1]}
		value=${BASH_REMATCH[2]}
		[[ -v PATTERNS[$key] ]] || die "$file, line $n: unknown key $key"
		[[ ! -v conf[$key] ]] || die "$file, line $n: key $key written twice"
		[[ $value =~ ${PATTERNS[$key]} ]] || die "$file, line $n: invalid value for $key"
		conf[$key]=$value
	done <"$file"
	for key in "${!PATTERNS[@]}"; do
		[[ -v conf[$key] ]] || die "$file: missing key $key"
	done
	((conf[SMTP_PORT] <= 65535)) || die "$file: invalid value for SMTP_PORT"
	[[ ${conf[ADMIN_USER]} != root && ${conf[ADMIN_USER]} != tribe-menus* ]] ||
		die "$file: ADMIN_USER must be a personal account"
	[[ ${conf[PRODUCTION_HOST]} != "${conf[RECETTE_HOST]}" ]] ||
		die "$file: PRODUCTION_HOST and RECETTE_HOST must differ"
}

# host_of prints the host name of an environment.
host_of() {
	case $1 in
	production) printf '%s' "${conf[PRODUCTION_HOST]}" ;;
	recette) printf '%s' "${conf[RECETTE_HOST]}" ;;
	esac
}

# --- Files made from the values ----------------------------------------------------------

# render_config prints the config.json of an environment (ADR 0015, point 9).
render_config() {
	local env=$1
	cat <<EOF
{
  "data": "/var/lib/tribe-menus/$env",
  "listen": "systemd",
  "baseURL": "https://$(host_of "$env")",
  "smtp": {
    "host": "${conf[SMTP_HOST]}",
    "port": ${conf[SMTP_PORT]},
    "username": "${conf[APP_SMTP_USER]}",
    "from": "${conf[APP_SMTP_FROM]}"
  },
  "alerts": {
    "to": "${conf[ADMIN_EMAIL]}"
  }
}
EOF
}

# Content-Security-Policy of the public site: HTML and CSS served by Caddy, without script
# nor third-party resource (plan recette, lot D).
readonly SITE_CSP="default-src 'none'; style-src 'self'; img-src 'self'; font-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

# render_caddyfile prints the Caddyfile of the environments given (ADR 0006; plan recette,
# step 12).
render_caddyfile() {
	local env host
	cat <<EOF
# Produced by provision.sh from $INSTANCE_CONF: do not edit (deploy/README.md).
{
	admin off
	email ${conf[ADMIN_EMAIL]}
	# Let's Encrypt only, the only authority the CAA record allows: no fallback to ZeroSSL.
	cert_issuer acme {
		dir https://acme-v02.api.letsencrypt.org/directory
	}
	servers {
		# No HTTP/3: the firewall opens no UDP port (plan recette, D16).
		protocols h1 h2
		timeouts {
			read_header 10s
			read_body 20s
			write 30s
			idle 2m
		}
		max_header_size 32KB
	}
}

(common) {
	header {
		Strict-Transport-Security "max-age=86400"
		-Server
	}
}

# The waiting page, in 503, when the socket of the application does not answer (no version
# installed, maintenance) or when no site is installed.
(waiting) {
	handle_errors 502 503 504 {
		import common
		root * $SHARE_DIR
		rewrite * /attente.html
		header Cache-Control no-store
		header Retry-After 300
		header Content-Security-Policy "default-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
		file_server {
			status 503
		}
	}
	handle_errors {
		import common
		header Cache-Control no-store
		respond "{err.status_code} {err.status_text}"
	}
}
EOF
	for env in "$@"; do
		host=$(host_of "$env")
		printf '\n%s {\n' "$host"
		cat <<'EOF'
	import common
	# Access logs to the journal (time, IP, URL), without the Cookie and Authorization
	# headers.
	log {
		format filter {
			request>headers>Cookie delete
			request>headers>Authorization delete
			resp_headers>Set-Cookie delete
		}
	}
	request_body {
		max_size 64KB
	}
EOF
		if [[ $env == recette ]]; then
			cat <<'EOF'
	# The recette is never indexed (ADR 0016, point 5).
	header X-Robots-Tag "noindex, nofollow"
	handle /robots.txt {
		respond "User-agent: *
Disallow: /
" 200
	}
EOF
		fi
		cat <<EOF

	# Only /tribes/* reaches the application: /healthz and every other path stay here.
	handle /tribes/* {
		# Host as received; X-Forwarded-For replaced by the address of the client, since no
		# proxy is trusted: the application reads its last address (ADR 0006, point 7).
		reverse_proxy unix//run/tribe-menus/$env.sock
	}

	# The public site of the environment (plan recette, D8).
	handle {
		root * /var/www/tribe-menus/$env
		@nosite not file {
			root /var/www/tribe-menus/$env
			try_files /index.html
		}
		error @nosite 503
		header {
			Content-Security-Policy "$SITE_CSP"
			X-Content-Type-Options nosniff
			Referrer-Policy same-origin
			Cross-Origin-Opener-Policy same-origin
		}
		file_server
	}

	import waiting
}
EOF
	done
}

# render_msmtprc prints the configuration of msmtp: the server sends with its own account
# (ADR 0015, point 6), whose password is decrypted from its credential at each sending.
render_msmtprc() {
	cat <<EOF
# Produced by provision.sh from $INSTANCE_CONF: do not edit (deploy/README.md).
defaults
auth on
tls on
tls_starttls off
tls_trust_file /etc/ssl/certs/ca-certificates.crt
syslog LOG_MAIL
aliases /etc/tribe-menus/aliases
# The sender is always the account of the server, which the MX Plan requires.
set_from_header on
allow_from_override off

account server
host ${conf[SMTP_HOST]}
port ${conf[SMTP_PORT]}
user ${conf[SERVER_SMTP_USER]}
from ${conf[SERVER_SMTP_FROM]}
passwordeval "systemd-creds decrypt --name=server-smtp-password /etc/tribe-menus/server/smtp-password.cred -"

account default : server
EOF
}

# render_aliases prints the aliases of msmtp: the mail of root and of any local account
# goes to the administrator.
render_aliases() {
	cat <<EOF
# Produced by provision.sh from $INSTANCE_CONF: do not edit (deploy/README.md).
root: ${conf[ADMIN_EMAIL]}
default: ${conf[ADMIN_EMAIL]}
EOF
}

render() {
	local dir=$1 env
	shift
	mkdir -p -- "$dir"
	for env in "$@"; do
		mkdir -p -- "$dir/$env"
		render_config "$env" >"$dir/$env/config.json"
	done
	render_caddyfile "$@" >"$dir/Caddyfile"
	render_msmtprc >"$dir/msmtprc"
	render_aliases >"$dir/aliases"
}

# --- System -----------------------------------------------------------------------------

# changed is set by put: 1 when the file was written, 0 when it was already up to date.
changed=0

# put writes stdin to a file, with its mode and owner, atomically, and only if its content,
# mode or owner differs.
put() {
	local mode=$1 owner=$2 dest=$3 tmp
	tmp=$(mktemp -- "$dest.XXXXXX")
	cat >"$tmp"
	chown -- "$owner" "$tmp"
	chmod -- "$mode" "$tmp"
	if [[ -f $dest ]] && cmp -s -- "$tmp" "$dest" &&
		[[ $(stat -c '%a %U:%G' -- "$tmp") == "$(stat -c '%a %U:%G' -- "$dest")" ]]; then
		rm -f -- "$tmp"
		changed=0
	else
		mv -f -- "$tmp" "$dest"
		changed=1
	fi
}

check_system() {
	[[ $(id -u) == 0 ]] || die "run as root: sudo $0 $*"
	if ! grep -qx 'ID=debian' /etc/os-release || ! grep -qx 'VERSION_ID="13"' /etc/os-release; then
		die "Debian 13 expected (ADR 0015, point 1)"
	fi
	[[ -d $DEPLOY_DIR/systemd && -d $DEPLOY_DIR/bin ]] || die "run from the deploy/ directory of an archive"
}

install_packages() {
	step "Packages"
	export DEBIAN_FRONTEND=noninteractive
	# Caddy from the repository of the Caddy project, its signing key in the archive
	# (plan recette, D15). The repository may only supply the caddy package.
	install -d -m 0755 /etc/apt/keyrings
	install -m 0644 -- "$DEPLOY_DIR/apt/caddy-stable.asc" /etc/apt/keyrings/caddy-stable.asc
	put 0644 root:root /etc/apt/sources.list.d/caddy-stable.sources <<'EOF'
# Produced by provision.sh: do not edit (deploy/README.md).
Types: deb
URIs: https://dl.cloudsmith.io/public/caddy/stable/deb/debian
Suites: any-version
Components: main
Signed-By: /etc/apt/keyrings/caddy-stable.asc
EOF
	put 0644 root:root /etc/apt/preferences.d/caddy-stable <<'EOF'
# Produced by provision.sh: do not edit (deploy/README.md).
# The repository of the Caddy project supplies caddy, and nothing else.
Package: *
Pin: origin dl.cloudsmith.io
Pin-Priority: -1

# Preferred to the older caddy of Debian.
Package: caddy
Pin: origin dl.cloudsmith.io
Pin-Priority: 600
EOF
	apt-get update -q
	apt-get install -y -q --no-install-recommends "${DEBIAN_PACKAGES[@]}" caddy
	# Time synchronisation: chrony if the image has it, systemd-timesyncd otherwise.
	if ! chrony_installed; then
		apt-get install -y -q --no-install-recommends systemd-timesyncd
	fi
}

chrony_installed() {
	dpkg-query -W -f='${Status}' chrony 2>/dev/null | grep -q 'ok installed'
}

configure_system() {
	step "Time, journal"
	timedatectl set-timezone Europe/Paris
	if chrony_installed; then
		systemctl enable --now --quiet chrony
	else
		timedatectl set-ntp true
	fi
	# Persistent journal, kept a month (PT-07), bounded.
	install -d -m 0755 /etc/systemd/journald.conf.d /var/log/journal
	put 0644 root:root /etc/systemd/journald.conf.d/50-tribe-menus.conf <<'EOF'
# Produced by provision.sh: do not edit (deploy/README.md).
[Journal]
Storage=persistent
SystemMaxUse=2G
MaxRetentionSec=1month
EOF
	if ((changed)); then
		systemctl restart systemd-journald
	fi

	step "Automatic security updates"
	put 0644 root:root /etc/apt/apt.conf.d/20auto-upgrades <<'EOF'
// Produced by provision.sh: do not edit (deploy/README.md).
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
EOF
	put 0644 root:root /etc/apt/apt.conf.d/51tribe-menus-unattended-upgrades <<'EOF'
// Produced by provision.sh: do not edit (deploy/README.md).
// Added to the patterns of 50unattended-upgrades of Debian, which installs the stable
// updates too: security fixes of Debian and the packages of the Caddy repository. A report
// by email to root when something changed, and a restart at 3:30 when one is needed, which
// the report announces and that shutdown -c cancels (ADR 0015, point 5).
Unattended-Upgrade::Origins-Pattern {
	"origin=Debian,codename=${distro_codename}-security,label=Debian-Security";
	"origin=cloudsmith/caddy/stable,codename=any-version";
};
Unattended-Upgrade::Mail "root";
Unattended-Upgrade::MailReport "on-change";
Unattended-Upgrade::Automatic-Reboot "true";
Unattended-Upgrade::Automatic-Reboot-WithUsers "true";
Unattended-Upgrade::Automatic-Reboot-Time "03:30";
Unattended-Upgrade::Remove-Unused-Kernel-Packages "true";
EOF
}

configure_mail() {
	step "Mail relay (msmtp)"
	install -d -m 0755 /etc/tribe-menus
	install -d -m 0700 /etc/tribe-menus/server
	render_msmtprc | put 0600 root:root /etc/msmtprc
	render_aliases | put 0644 root:root /etc/tribe-menus/aliases
	# Key of the machine, which encrypts the credentials (ADR 0015, point 10).
	systemd-creds setup
}

# configure_ssh creates the personal administration account and closes SSH to anything but
# its key (ADR 0015, point 3; plan recette, D14). Root keeps its key until the script runs
# from that account through sudo, which proves the account works (ADR 0025, point 3).
configure_ssh() {
	local admin=${conf[ADMIN_USER]} root_login allow home
	local drop_in=/etc/ssh/sshd_config.d/00-tribe-menus.conf
	step "Administration account and SSH"
	if ! id -u -- "$admin" >/dev/null 2>&1; then
		useradd --create-home --shell /bin/bash --groups sudo -- "$admin"
	fi
	usermod --append --groups sudo -- "$admin"
	home=$(getent passwd -- "$admin" | cut -d: -f6)
	if [[ ! -s $home/.ssh/authorized_keys ]]; then
		[[ -s /root/.ssh/authorized_keys ]] || die "no SSH key for $admin, and none for root to copy"
		install -d -m 0700 -o "$admin" -g "$admin" -- "$home/.ssh"
		install -m 0600 -o "$admin" -g "$admin" -- /root/.ssh/authorized_keys "$home/.ssh/authorized_keys"
	fi
	# sudo asks for a password (D14): the account must have one.
	if [[ $(passwd --status -- "$admin" | cut -d' ' -f2) != P ]]; then
		[[ -t 0 ]] || die "$admin has no password, which sudo asks for: run from a terminal"
		printf 'Password of %s, asked by sudo (keep it in the password manager):\n' "$admin"
		passwd -- "$admin"
	fi
	if grep -rEqs '^[^#]*NOPASSWD' /etc/sudoers /etc/sudoers.d; then
		printf 'provision: warning: a sudoers rule grants sudo without a password:\n' >&2
		grep -rEls '^[^#]*NOPASSWD' /etc/sudoers /etc/sudoers.d >&2 || true
	fi

	install -d -m 0700 -- "$STATE_DIR"
	if [[ ${SUDO_USER:-} == "$admin" ]]; then
		touch -- "$ROOT_SSH_CLOSED"
	fi
	if [[ -e $ROOT_SSH_CLOSED ]]; then
		root_login=no allow=$admin
	else
		root_login=prohibit-password allow="$admin root"
	fi
	# First match wins in sshd_config: this file, read first, prevails over the image's. The
	# previous one is kept until sshd accepts the new one: a refused file would lock SSH at
	# the next restart.
	rm -f -- "$drop_in.previous"
	if [[ -f $drop_in ]]; then
		cp -p -- "$drop_in" "$drop_in.previous"
	fi
	put 0644 root:root "$drop_in" <<EOF
# Produced by provision.sh: do not edit (deploy/README.md).
PermitRootLogin $root_login
AllowUsers $allow
AuthenticationMethods publickey
PubkeyAuthentication yes
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitEmptyPasswords no
X11Forwarding no
AllowAgentForwarding no
AllowTcpForwarding no
AllowStreamLocalForwarding no
PermitTunnel no
EOF
	if ((changed)); then
		if ! sshd -t; then
			if [[ -f $drop_in.previous ]]; then
				mv -f -- "$drop_in.previous" "$drop_in"
			else
				rm -f -- "$drop_in"
			fi
			die "sshd refuses its configuration: previous one put back, the running daemon is unchanged"
		fi
		systemctl reload ssh
	fi
	rm -f -- "$drop_in.previous"
	# sshd -T writes prohibit-password under its older name.
	sshd -T | grep -Eqx "permitrootlogin ${root_login/prohibit-password/(prohibit|without)-password}" ||
		die "sshd does not apply PermitRootLogin $root_login"
	sshd -T | grep -qx 'passwordauthentication no' || die "sshd does not apply PasswordAuthentication no"
	if [[ $root_login != no ]]; then
		printf '\nRoot can still connect by SSH, with its key. In a second terminal, connect as %s,\n' "$admin"
		printf 'then run this script again with sudo from that account: it closes SSH to root.\n'
	fi
}

configure_firewall() {
	step "Firewall (nftables)"
	# TCP 22, 80 and 443 in, the ICMP and ICMPv6 that IPv4 and IPv6 need, no UDP port
	# (ADR 0015, point 4; plan recette, D16). Replies go through the connection tracking.
	put 0755 root:root /etc/nftables.conf <<'EOF'
#!/usr/sbin/nft -f
# Produced by provision.sh: do not edit (deploy/README.md).
flush ruleset

table inet filter {
	chain input {
		type filter hook input priority filter; policy drop;
		ct state established,related accept
		ct state invalid drop
		iif lo accept
		icmp type { destination-unreachable, time-exceeded, parameter-problem, echo-request } accept
		icmpv6 type { destination-unreachable, packet-too-big, time-exceeded, parameter-problem, echo-request } accept
		icmpv6 type { nd-router-advert, nd-neighbor-solicit, nd-neighbor-advert } ip6 hoplimit 255 accept
		# Queries of the router, without which switches that snoop MLD stop forwarding the
		# neighbour solicitations (RFC 4890, 4.4.1; review of PR #55, point 2).
		icmpv6 type mld-listener-query ip6 saddr fe80::/10 accept
		tcp dport { 22, 80, 443 } accept
	}
	chain forward {
		type filter hook forward priority filter; policy drop;
	}
	chain output {
		type filter hook output priority filter; policy accept;
	}
}
EOF
	nft --check --file /etc/nftables.conf || die "nft refuses /etc/nftables.conf"
	systemctl enable --quiet nftables
	# Never restart: the unit of Debian flushes the rules on stop, leaving the input open
	# until they are loaded again. Its reload runs nft -f on the file, whose flush ruleset
	# and rules apply in a single transaction: the established connections, this SSH
	# session among them, are kept (review of PR #55, point 1).
	if ! systemctl is-active --quiet nftables; then
		systemctl start nftables
	elif ((changed)); then
		systemctl reload nftables
	fi
}

install_files() {
	local script
	step "Scripts and units"
	for script in "${SCRIPTS[@]}"; do
		install -m 0755 -- "$DEPLOY_DIR/bin/$script" "$BIN_DIR/$script"
	done
	install -d -m 0755 -- "$SHARE_DIR"
	install -m 0644 -- "$DEPLOY_DIR/caddy/attente.html" "$SHARE_DIR/attente.html"
	install -m 0644 -- "$DEPLOY_DIR/systemd/tribe-menus@.socket" /etc/systemd/system/tribe-menus@.socket
	install -m 0644 -- "$DEPLOY_DIR/systemd/tribe-menus@.service" /etc/systemd/system/tribe-menus@.service
	systemctl daemon-reload
}

# configure_environment creates the user, the directories and the configuration of an
# environment (ADR 0015, points 8 and 9; plan recette, D8). An instance reads nothing of
# the other: its data and configuration belong to its user.
configure_environment() {
	local env=$1 user=tribe-menus-$1
	step "Environment $env"
	if ! id -u -- "$user" >/dev/null 2>&1; then
		useradd --system --user-group --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin -- "$user"
	fi
	install -d -m 0755 /opt/tribe-menus "/opt/tribe-menus/$env" "/opt/tribe-menus/$env/releases"
	install -d -m 0750 -o root -g "$user" "/etc/tribe-menus/$env"
	install -d -m 0755 /var/lib/tribe-menus
	install -d -m 0700 -o "$user" -g "$user" "/var/lib/tribe-menus/$env"
	install -d -m 0755 /var/www/tribe-menus "/var/www/tribe-menus/$env"
	render_config "$env" | put 0640 "root:$user" "/etc/tribe-menus/$env/config.json"
	systemctl enable --quiet "tribe-menus@$env.socket"
	if ((changed)); then
		systemctl try-restart "tribe-menus@$env.service"
	fi
	# Listens only once a version, the configuration and the credential are there.
	systemctl start "tribe-menus@$env.socket"
}

configure_caddy() {
	step "Caddy"
	render_caddyfile "$@" | put 0644 root:root /etc/caddy/Caddyfile
	runuser -u caddy -- caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile >/dev/null ||
		die "caddy refuses /etc/caddy/Caddyfile"
	systemctl enable --quiet caddy
	# With admin off, caddy reload has no API to talk to: restart.
	if ((changed)) || ! systemctl is-active --quiet caddy; then
		systemctl restart caddy
	fi
}

# summary says what the server still lacks to serve each environment.
summary() {
	local env missing
	step "Still missing"
	missing=0
	[[ -s /etc/tribe-menus/server/smtp-password.cred ]] ||
		{ printf '  server: password of %s (sudo tribe-menus-credential server smtp-password)\n' "${conf[SERVER_SMTP_USER]}"; missing=1; }
	[[ -s /etc/tribe-menus/server/github-token.cred ]] ||
		{ printf '  server: GitHub token (sudo tribe-menus-credential server github-token)\n'; missing=1; }
	for env in "$@"; do
		[[ -s /etc/tribe-menus/$env/smtp-password.cred ]] ||
			{ printf '  %s: SMTP password (sudo tribe-menus-credential %s smtp-password)\n' "$env" "$env"; missing=1; }
		[[ -x /opt/tribe-menus/$env/current/tribe-menus ]] ||
			{ printf '  %s: no version installed (tribe-menus-deploy)\n' "$env"; missing=1; }
	done
	((missing)) || printf '  nothing\n'
}

main() {
	local render_dir="" instance=$INSTANCE_CONF env
	while (($#)); do
		case $1 in
		--render)
			(($# >= 2)) || usage
			render_dir=$2
			shift 2
			;;
		--instance)
			(($# >= 2)) || usage
			instance=$2
			shift 2
			;;
		-*) usage ;;
		*) break ;;
		esac
	done
	(($#)) || usage
	for env in "$@"; do
		[[ $env == production || $env == recette ]] || usage
	done
	if [[ -n $render_dir ]]; then
		read_instance "$instance"
		render "$render_dir" "$@"
		return
	fi
	[[ $instance == "$INSTANCE_CONF" ]] || usage

	check_system "$@"
	read_instance "$INSTANCE_CONF"
	install_packages
	configure_system
	configure_mail
	configure_ssh
	configure_firewall
	install_files
	for env in "$@"; do
		configure_environment "$env"
	done
	configure_caddy "$@"
	summary "$@"
}

main "$@"
