FROM ubuntu:latest
COPY sointu-server .
RUN chmod a+x /sointu-server
CMD /sointu-server