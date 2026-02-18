# GoReleaser puts only the pre-built binary (and this Dockerfile) in the build context.
# The binary is built with embedded UI earlier in the release; we just copy it.
FROM gcr.io/distroless/static:nonroot

COPY agentary /agentary

EXPOSE 3548

USER nonroot:nonroot

ENV AGENTARY_HOME=/data
ENTRYPOINT ["/agentary"]
CMD ["start", "--foreground", "--home", "/data"]
