FROM debian:bullseye as tmux
ENV DEBIAN_FRONTEND=noninteractive
# Set the working directory
WORKDIR /tmux
# Install dependencies
RUN apt-get update && apt-get install -y \
  git \
  automake \
  build-essential \
  pkg-config \
  libevent-dev \
  libncurses5-dev \
  bison
# Clone the tmux repository and checkout version 3.4(Versions can be switched)
RUN git clone https://github.com/tmux/tmux.git . && git checkout 3.5a
# Generate the configure script
RUN sh autogen.sh
# Configure and build tmux
RUN ./configure && make

FROM docker.io/golang:1.23
COPY --from=tmux /tmux/tmux /usr/bin/tmux
COPY post-install.sh /tmp/post-install.sh
RUN bash /tmp/post-install.sh 
