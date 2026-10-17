#include <iostream>
#include <cstdlib>
#include <string>
#include <limits>
#include <vector>
#include <sstream>
#include <numeric>
#include <ctime>
#include <cmath>

// just a random comment
using namespace std; // replace the repitative std
int imGlobal = 12;
const double PI = 3.14;
const string name = "sri";
const char initial = 's';
int main(){
    cout << "Feuille Labs" << endl;
    bool married = false;
    char myGrade = 'A';
    double grade = 100;
    double extra = 15.25;
    printf("Sum = %.7f\n", (grade+extra));
    auto Num1 = 2;
    cout << Num1 << endl << "The size of an integer: " << sizeof(Num1);
    printf("%c %d %5d %.3f &s\n", 'A', 10,5,3.123,"Hi");
    return 0;
}
