# Image du Dev Container (ADR 0009, 0018).
# Aucune version d'outil du projet n'est dupliquée ici : Go suit la directive
# `toolchain` de go.mod (téléchargée au besoin), Node suit .nvmrc. Seule la
# version de Claude Code est épinglée ici (et dans devcontainer.json).
FROM golang:1.27-trixie

ARG USERNAME=vscode
ARG USER_UID=1000
ARG CLAUDE_CODE_VERSION=2.1.284

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

# Claude Code, version épinglée (installateur officiel). Le dossier de
# configuration est créé ici pour que le volume monté dessus appartienne à
# l'utilisateur.
RUN mkdir -p "/home/${USERNAME}/.claude" \
 && curl -fsSL https://claude.ai/install.sh | bash -s "${CLAUDE_CODE_VERSION}" \
 && claude --version
