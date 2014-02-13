
FROM ubuntu:12.04
MAINTAINER Spyros Anastasopoulos <winwasher@gmail.com>

# set ports to publicly expose when running the image
EXPOSE 8000 8001


# add copies files from host to container. Supports dirs, tgz, http
ADD ./webserver /home/webserver

# set cwd
WORKDIR /home


# set environment variables for RUN commands
ENV mode dev

# prepare the container. ex apt-get update
#RUN

# main executable
#CMD
#ENTRYPOINT

#
# technicalities
#

# add volumes to containers created by the image
#VOLUME ["/data"]

# sets username or UID when running the image
#USER spyros
