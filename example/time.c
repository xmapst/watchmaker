#include <stdio.h>
#include <unistd.h>
#include <time.h>

int main() {
    time_t current_time;

    while (1) {
        current_time = time(NULL);
        if (current_time == -1) {
            perror("time() failed");
            return 1;
        }

        printf("[%d] time() returns: %s", getpid(), ctime(&current_time));

        sleep(1);
    }

    return 0;
}
