JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-amd64/
INCLUDE=../src
APPLICATION=simple
JAVADIR=java
SCALADIR=scala

.PHONY: clean java scala

all: java scala

clean:
	rm -f tracer.o libjnitracer.so $(APPLICATION).class


tracer.o: tracer.c
	gcc -c -fPIC -I${JAVA_HOME}/include -I${JAVA_HOME}/include/linux -I${INCLUDE} ${JAVAOPT} $<  -o $@

libjnitracer.so: tracer.o ../lib/libtracer.a
	gcc -shared -fPIC -o $@ -I${INCLUDE} tracer.o -Wl,--whole-archive,-static -L../lib -ltracer -Wl,--no-whole-archive,-Bdynamic -lc

$(JAVADIR)/$(APPLICATION).class: $(JAVADIR)/$(APPLICATION).java
	javac -d $(JAVADIR) $<

$(SCALADIR)/$(APPLICATION).class: $(SCALADIR)/$(APPLICATION).scala
	scalac -d $(SCALADIR) $<

scala: $(SCALADIR)/$(APPLICATION).class libjnitracer.so 
	cd $(SCALADIR) ; scala -cp . -Djava.library.path=../ TracerTest

java: $(JAVADIR)/$(APPLICATION).class libjnitracer.so
	cd $(JAVADIR) ; java -cp . -Djava.library.path=../ $(APPLICATION)

