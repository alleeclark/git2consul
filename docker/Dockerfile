FROM golang

RUN apt-get update && apt-get -q -y install \
	git openssl apt-transport-https ca-certificates curl g++ gcc libc6-dev make pkg-config \
	libssl-dev cmake

# Build libssh2 from source
RUN cd $HOME && curl -fsSL https://github.com/libssh2/libssh2/archive/libssh2-1.8.2.tar.gz -o libssh2.tar.gz \
    && mkdir libgit2 \
 	&& tar xvf libssh2.tar.gz -C libgit2 \
	&& ls -la libgit2 \
	&& cd libgit2/libssh2-libssh2-1.8.2 \
	&& cmake -DBUILD_SHARED_LIBS=ON . \
	&& cmake --build . \
	&& make \
	&& make install \
	&& ldconfig

# Build libgit2 from source
RUN cd $HOME && curl -fsSL https://github.com/libgit2/libgit2/releases/download/v0.99.0/libgit2-0.99.0.tar.gz -o v0.99.0.tar.gz \
 	&& tar xvf v0.99.0.tar.gz -C libgit2 \
	&& cd libgit2/libgit2-0.99.0 \
	&& cmake -DCURL=OFF . \
	&& cmake --build . \
	&& make \
	&& make install \
	&& ldconfig \
	&& rm -rf $HOME/libgit2

WORKDIR ${GOPATH}/src/
RUN mkdir git2consul
COPY . ${GOPATH}/src/git2consul
WORKDIR ${GOPATH}/src/git2consul
RUN go mod download
RUN go install
ENTRYPOINT [ "git2consul" ]