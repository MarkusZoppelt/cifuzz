package com.example;

import com.code_intelligence.jazzer.api.FuzzedDataProvider;
import com.code_intelligence.jazzer.junit.FuzzTest;

public class FuzzTestCase {
    @FuzzTest
    void oneFuzzTest(FuzzedDataProvider data) {
        int a = data.consumeInt();
        int b = data.consumeInt();
        String c = data.consumeRemainingAsString();

        ExploreMe ex = new ExploreMe(a);
        ex.exploreMe(b, c);
    }

    @FuzzTest
    void anotherFuzzTest(FuzzedDataProvider data) {
        int a = data.consumeInt();
        int b = data.consumeInt();
        String c = data.consumeRemainingAsString();

        ExploreMe ex = new ExploreMe(a);
        ex.exploreMe(b, c);
    }
}
