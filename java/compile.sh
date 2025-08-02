gcc -fpic -I/usr/lib/jvm/java-1.8.0-openjdk-amd64/include -I/usr/lib/jvm/java-1.8.0-openjdk-amd64/include/linux/ -I./ -c A.c -o A.o

gcc -shared -o libHS.so A.o -Wl,-rpath,/home/jingyuan/hindsight/client/lib -L/home/jingyuan/hindsight/client/lib/ -ltracer
