JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-amd64/

all: run

simple.class: simple.java
	javac $<

tracer.o: tracer.cpp
	g++ -c -fPIC -I${JAVA_HOME}/include -I${JAVA_HOME}/include/linux ${JAVAOPT} $<  -o $@

libtracer.so: tracer.o
	g++ -shared -fPIC -o $@ $< -lc

run: simple.class libtracer.so
	java -cp . -Djava.library.path=./ simple

