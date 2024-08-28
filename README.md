# solana-lsd-relay

Given the limitation of smart contracts, they are not self-executing, and requiring an external force to trigger their functions. Solana LSD Relay is an offchain service that drives LSD network to properly process its own internal states, such as dealing with delegating, undelegating, distributing rewards and recalculating the rate between LST and SOL. We introduce `era` concept to define how often in epochs the smart contract should be called.

*Security* is the our first priority when developing Stack, We are thrilled to say that the new era process is permissionless, showcasing the decentralized nature of the StaFi SOL LSD Stack, allowing anyone to trigger the new era process.

To learn more about Solana LSD stack, see [Documentation and Guide](https://lsaas-docs.stafi.io/docs/architecture/solana_lsd.html)
