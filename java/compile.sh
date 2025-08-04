gcc -fpic -I/usr/lib/jvm/java-1.8.0-openjdk-amd64/include -I/usr/lib/jvm/java-1.8.0-openjdk-amd64/include/linux/ -I./ -I/home/jingyuan/hindsight/client/include -I/home/jingyuan/hindsight/client/src -c HindSightJNI.c -o HindSightJNI.o

gcc -shared -o libHS.so HindSightJNI.o -Wl,-rpath,/home/jingyuan/hindsight/client/lib -L/home/jingyuan/hindsight/client/lib/ -ltracer
