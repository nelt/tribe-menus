# Image du Dev Container (ADR 0009, 0018, 0019).
# Aucune version d'outil du projet n'est dupliquée ici : Go suit la directive
# `toolchain` de go.mod (téléchargée au besoin), Node suit .nvmrc. Seules les
# versions de Claude Code (aussi dans devcontainer.json) et de GitHub CLI sont
# épinglées ici ; Dependabot ne les suit pas, montée de version par PR.
FROM golang:1.27-trixie

ARG USERNAME=vscode
ARG USER_UID=1000
ARG CLAUDE_CODE_VERSION=2.1.284
ARG GH_VERSION=2.102.0

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Paquets système : client sqlite3 (ADR 0009), sudo (installation des
# dépendances système de Playwright par make tools).
RUN apt-get update \
 && apt-get install -y --no-install-recommends sqlite3 sudo xz-utils \
 && rm -rf /var/lib/apt/lists/*

# Node à la version de .nvmrc, archive officielle vérifiée par son empreinte.
COPY .nvmrc /tmp/.nvmrc
RUN version="$(tr -d '[:space:]v' < /tmp/.nvmrc)" \
 && case "$(dpkg --print-architecture)" in \
      amd64) arch=x64 ;; \
      arm64) arch=arm64 ;; \
      *) echo "architecture non prise en charge" >&2; exit 1 ;; \
    esac \
 && file="node-v${version}-linux-${arch}.tar.xz" \
 && cd /tmp \
 && curl -fsSLO "https://nodejs.org/dist/v${version}/${file}" \
 && curl -fsSL "https://nodejs.org/dist/v${version}/SHASUMS256.txt" | grep " ${file}\$" | sha256sum -c - \
 && tar -xJf "${file}" -C /usr/local --strip-components=1 --no-same-owner \
 && rm -f "${file}" /tmp/.nvmrc \
 && node --version

# GitHub CLI, en lecture seule sur le dépôt (ADR 0019) : archive officielle
# vérifiée par l'empreinte publiée avec la version.
RUN case "$(dpkg --print-architecture)" in \
      amd64|arm64) arch="$(dpkg --print-architecture)" ;; \
      *) echo "architecture non prise en charge" >&2; exit 1 ;; \
    esac \
 && file="gh_${GH_VERSION}_linux_${arch}.tar.gz" \
 && cd /tmp \
 && curl -fsSLO "https://github.com/cli/cli/releases/download/v${GH_VERSION}/${file}" \
 && curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_checksums.txt" | grep " ${file}\$" | sha256sum -c - \
 && tar -xzf "${file}" -C /usr/local --strip-components=1 --no-same-owner "gh_${GH_VERSION}_linux_${arch}/bin/gh" \
 && rm -f "${file}" \
 && gh --version

# Utilisateur non privilégié. Son UID est aligné sur celui de l'hôte à la
# création du conteneur (updateRemoteUserUID), pour les droits sur le dépôt.
RUN useradd --create-home --shell /bin/bash --uid "${USER_UID}" "${USERNAME}" \
 && echo "${USERNAME} ALL=(root) NOPASSWD:ALL" > "/etc/sudoers.d/${USERNAME}" \
 && chmod 0440 "/etc/sudoers.d/${USERNAME}"

USER ${USERNAME}
ENV GOPATH=/home/${USERNAME}/go
# L'image officielle fixe GOTOOLCHAIN=local ; on rétablit le comportement par
# défaut pour que la directive `toolchain` de go.mod s'applique (ADR 0009).
ENV GOTOOLCHAIN=auto
ENV PATH=/home/${USERNAME}/.local/bin:/home/${USERNAME}/go/bin:${PATH}

# Claude Code, version épinglée (installateur officiel). Les dossiers de
# configuration de Claude Code et de GitHub CLI sont créés ici pour que les
# volumes montés dessus appartiennent à l'utilisateur.
RUN mkdir -p "/home/${USERNAME}/.claude" "/home/${USERNAME}/.config/gh" \
 && curl -fsSL https://claude.ai/install.sh | bash -s "${CLAUDE_CODE_VERSION}" \
 && claude --version
