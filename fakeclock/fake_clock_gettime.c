#include <time.h>
#include <inttypes.h>
#include <syscall.h>

extern int64_t TV_SEC_DELTA;
extern int64_t TV_NSEC_DELTA;
extern uint64_t CLOCK_IDS_MASK;

#if defined(__amd64__)
inline int real_clock_gettime(clockid_t clk_id, struct timespec *tp) {
    int ret;
    asm volatile
        (
            "syscall"
            : "=a" (ret)
            : "0"(__NR_clock_gettime), "D"(clk_id), "S"(tp)
            : "rcx", "r11", "memory"
        );
    return ret;
}
#elif defined(__aarch64__)
inline int real_clock_gettime(clockid_t clk_id, struct timespec *tp) {
    register clockid_t x0 __asm__ ("x0") = clk_id;
    register struct timespec *x1 __asm__ ("x1") = tp;
    register uint64_t w8 __asm__ ("w8") = __NR_clock_gettime;
    __asm__ __volatile__ (
        "svc 0"
        : "+r" (x0)
        : "r" (x0), "r" (x1), "r" (w8)
        : "memory"
    );
    return x0;
}
#endif

int fake_clock_gettime(clockid_t clk_id, struct timespec *tp) {
    int ret = real_clock_gettime(clk_id, tp);

    const int64_t sec_delta = TV_SEC_DELTA;
    const int64_t nsec_delta = TV_NSEC_DELTA;
    const uint64_t clock_ids_mask = CLOCK_IDS_MASK;

    const uint64_t clk_id_mask = UINT64_C(1) << clk_id;
    if ((clk_id_mask & clock_ids_mask) != 0) {
        const int64_t billion = 1000000000;

        int64_t sec = tp->tv_sec;
        int64_t nsec = tp->tv_nsec;

        int64_t total_nsec = nsec + nsec_delta;
        int64_t extra_sec = total_nsec / billion;
        int64_t adjusted_nsec = total_nsec % billion;

        if (adjusted_nsec < 0) {
            extra_sec -= 1;
            adjusted_nsec += billion;
        }

        sec += sec_delta + extra_sec;
        nsec = adjusted_nsec;

        tp->tv_sec = sec;
        tp->tv_nsec = nsec;
    }

    return ret;
}