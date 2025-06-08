#include <stdio.h>
#include <unistd.h>
#include <time.h>

int main() {
    time_t current_time;

    for (int i = 0; i < 10; i++) {
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
