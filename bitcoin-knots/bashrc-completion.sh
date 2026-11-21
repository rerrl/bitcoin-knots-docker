# Enable bash completion for interactive non-login shells (e.g. `docker exec -it bitcoind bash`),
# which read /etc/bash.bashrc rather than the login-only /etc/profile.d scripts.
if ! shopt -oq posix && [ -z "${BASH_COMPLETION_VERSINFO+set}" ]; then
    if [ -r /usr/share/bash-completion/bash_completion ]; then
        . /usr/share/bash-completion/bash_completion
    fi
fi
