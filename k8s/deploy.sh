#!/usr/bin/env bash

# should have the following environment variables set
# PORT, ENVIRONMENT, DISCORD_TOKEN, AMQP_DSN, LOGGING_DISCORD_WEBHOOK, DOCKER_IMAGE

manifest=$(envsubst < ./k8s/manifest.tmpl.yaml)

applyCommand="kubectl"
# applyCommand+=" --kubeconfig ~/Downloads/devtest-kubeconfig.yaml"
applyCommand+=" apply -f -"
# applyCommand+=" delete -f -"
applyCommand+=" <<EOF"
applyCommand+=$'\n'
applyCommand+="$manifest"
applyCommand+=$'\n'
applyCommand+="EOF"

echo "$applyCommand"
eval "$applyCommand"