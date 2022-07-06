---
title: IUntroduction to Clef
sort_key: A
---


  
## What is Clef?

Clef is a tool for signing transactions and data. It is intended to become a more secure replacement 
for Geth's built-in account management. This decouples key management from Geth itself, providing a
more modular and flexible tool compared to Geth's account manager. Clef can be used safely in situations 
where access to Ethereum is via a remote and/or untrusted node because signing happens locally under custom 
rulesets. The separation of Clef from the node itself enables Clef to run as a daemon on the same machine as the
client software, on a secure usb-stick like USB armory, or even a separate VM in a QubesOS type setup.

{:toc}

-   this will be removed by the toc
  
