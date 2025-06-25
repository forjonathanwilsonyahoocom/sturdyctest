# sturdyctest
a test of sturdy c batch get

this test makes arbitrary number of batches of arbitrary size where the keys are almost completly overlapping

ie this is the perfect stress test for this tool

as currently written it runs with some obvious contention on my box, reducing the batch size by a power of ten seems to work just fine, increasing by a power of ten crashes reliably
