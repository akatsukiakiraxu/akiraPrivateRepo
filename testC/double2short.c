#include <stdio.h>
void dump (unsigned char *p, int n);
void main() {
	int i;
	union {
		double d;
		unsigned char c[8];
	} uni;

 	printf("number: ");
	scanf("%lf", &uni.d);
	printf("d = %lf\n", uni.d);
	dump(uni.c, 8);
	
	union {
	    float f;
	    unsigned char c[4];
	} unif;
	
	unif.f = (float)uni.d;
 	printf("float: %f\n", unif.f);
	dump(unif.c, 4);

}        
void dump(unsigned char *p, int n) {
	int i;
	
	for (i=n-1;i>=0;i--) printf("%02X ", p[i]);
	printf("\n");
}
