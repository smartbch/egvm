#include <stdint.h>

#define RETRY_LIMIT 10

#define DRNG_SUCCESS 1
#define DRNG_NOT_READY -1

#if defined(__APPLE__)
    #define _rdrand_step(x) ({ unsigned char err; err;})
#else
    #define _rdrand_step(x) ({ unsigned char err; asm volatile("rdrand %0; setc %1":"=r"(*x), "=qm"(err)); err; })
#endif

#define _rdrand16_step(x) _rdrand_step(x)

int rdrand16(uint16_t* x, int retry)
{
	unsigned int i;
		if (retry)
		{
			for (i = 0; i < RETRY_LIMIT; i++)
			{
				if (_rdrand16_step(x))
					return DRNG_SUCCESS;
			}

			return DRNG_NOT_READY;
		}
		else
		{
				if (_rdrand16_step(x))
					return DRNG_SUCCESS;
				else
					return DRNG_NOT_READY;
		}
}

