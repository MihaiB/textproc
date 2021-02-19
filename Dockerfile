FROM docker.io/library/golang
RUN ["useradd", "--create-home", "user"]
USER user:user
RUN ["mkdir", "/home/user/textproc"]
WORKDIR /home/user/textproc
COPY --chown=user:user [".", "."]
RUN ["./docker.sh"]
