## An algorithm for rate limiting with exponential probabalistic decay

/**
 * @author dhanji@gmail.com (Dhanji R. Prasanna)
 */
private static final long THRESHOLD = 10; // 10 calls per millisecond
private static final long OPEN_FLOW = THRESHOLD * 0.5;
private static final long SAMPLE_RATE = 100; // millis

private volatile long lastCallAt = System.currentTimeMillis();
private volatile long rate;
private volatile long count;

// This can be made much faster by using thread-local random number generators.
private final Random random = new Random(lastCallAt);

public void throttle(Invocation inv) {
  long diff = System.currentTimeMillis() - lastCallAt;
  count++; // Not atomic, but OK: accuracy is traded for speed.

  // Sample invocation rate every n millis.
  if (diff > SAMPLE_RATE) {
    rate = count / diff;
    count = 0;
    lastCallAt = System.currentTimeMillis();
  }

  // Upto 50% of threshold, we allow calls unfettered.
  if (rate < OPEN_FLOW)
    inv.proceed();
  else {  // Might want to complement this with a rate limit that drops ALL invocations.

    // Fuzz factor: the probability of proceeding decreases as burst approaches
    // 1. So there is a 75% chance to proceed at burst of 50%; a 44% chance at
    // burst of 75%; 19% at burst of 90%, etc. This gives you fuzzed exponential
    // falloff.
    // P(proceed) = 1 - burst^2

    double burst = rate / THRESHOLD;
    if (random.nextDouble() > burst * burst)
      inv.proceed();
  }
}
