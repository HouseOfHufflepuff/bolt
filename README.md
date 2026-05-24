# Bolt

Asset issuance platform on Bitcoin Lightning using the Taproot Assets Protocol. Issues BOLT — a basket currency backed by USD, CHF, and BTC — and the constituent assets individually.

---

## BOLT Token

BOLT is a basket-of-assets currency. One BOLT is a redeemable claim on a weighted pool of:

| Asset | Initial Weight | Issued As |
|---|---|---|
| USD | 40% | USDL |
| CHF | 40% | CHFL |
| BTC | 20% | Native (Lightning) |

**Weights are fixed per epoch and rebalance periodically.** Price floats with the market. Each constituent asset is individually redeemable via API.

### Why CHF over EUR

The Swiss National Bank (SNB) is constitutionally mandated for price stability with zero political override history. CHF averages 1.4% inflation over 5 years vs. USD at 4.5%. It is the strongest long-term purchasing power preserver among major liquid currencies. EUR is excluded — structurally expansionary monetary union, no coherent single mandate.
