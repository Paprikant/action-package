FROM ubuntu:20.04

RUN \
  apt-get -y update 					 	&&\
  apt-get install -y ruby ruby-dev rubygems build-essential 	&&\
  gem install fpm                                               &&\
  apt-get remove -y ruby-dev rubygems                           &&\
  apt-get -y autoremove                                         &&\
  apt-get -qq clean

# provide the build-packages binary and use it as entrypoint
COPY build-packages /build-packages
ENTRYPOINT ["/build-packages"]
