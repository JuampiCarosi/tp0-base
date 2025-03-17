# syntax=docker/dockerfile:1.7-labs

FROM python:3.9.7-slim
COPY --exclude=server/config.ini server /
RUN python -m unittest tests/test_common.py
ENTRYPOINT ["/bin/sh"]
