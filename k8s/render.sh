#!/usr/bin/env bash

# should have the following environment variables set:
# DISCORD_TOKENS
# AMQP_DSN
# LOGGING_DISCORD_WEBHOOK
# PORT
# ENVIRONMENT
# DOCKER_IMAGE_HASH
# REDIS_ADDRESS
# REDIS_PASSWORD
# ENABLE_WHITELIST
# ERRORTRACKING_RAVEN_DSN
# HASH
# CLUSTER_ENVIRONMENT
# DISCORD_API_BASE
# DEDUPLICATE
# REQUEST_MEMBERS_DELAY
# HONEYCOMB_API_KEY

template="k8s/manifest.tmpl.yaml"
target="k8s/manifest.yaml"

cp "$template" "$target"
sed -i -e "s|{{DISCORD_TOKENS}}|$DISCORD_TOKENS|g" "$target"
sed -i -e "s|{{AMQP_DSN}}|$AMQP_DSN|g" "$target"
sed -i -e "s|{{LOGGING_DISCORD_WEBHOOK}}|$LOGGING_DISCORD_WEBHOOK|g" "$target"
sed -i -e "s|{{PORT}}|$PORT|g" "$target"
sed -i -e "s|{{ENVIRONMENT}}|$ENVIRONMENT|g" "$target"
sed -i -e "s|{{DOCKER_IMAGE_HASH}}|$DOCKER_IMAGE_HASH|g" "$target"
sed -i -e "s|{{REDIS_ADDRESS}}|$REDIS_ADDRESS|g" "$target"
sed -i -e "s|{{REDIS_PASSWORD}}|$REDIS_PASSWORD|g" "$target"
sed -i -e "s|{{ENABLE_WHITELIST}}|$ENABLE_WHITELIST|g" "$target"
sed -i -e "s|{{ERRORTRACKING_RAVEN_DSN}}|$ERRORTRACKING_RAVEN_DSN|g" "$target"
sed -i -e "s|{{HASH}}|$HASH|g" "$target"
sed -i -e "s|{{CLUSTER_ENVIRONMENT}}|$CLUSTER_ENVIRONMENT|g" "$target"
sed -i -e "s|{{DISCORD_API_BASE}}|$DISCORD_API_BASE|g" "$target"
sed -i -e "s|{{DEDUPLICATE}}|$DEDUPLICATE|g" "$target"
sed -i -e "s|{{REQUEST_MEMBERS_DELAY}}|$REQUEST_MEMBERS_DELAY|g" "$target"
sed -i -e "s|{{HONEYCOMB_API_KEY}}|$HONEYCOMB_API_KEY|g" "$target"
