## Architecture

> Paraphrasing from some guy's blog, "architecture" is simply what is *most important*, and must be kept in good condition in a system's design.

The code itself is organized in broad "layers", what determines where a layer is located is *how prone its interface is to change*. Layers that are higher up, are more prone to change, layers lower are less.

Layers of abstraction as a concept itself is relatively simple, each layer depends on the interface(s) given by the layer beneath it and doesn't care exactly what it needs to do in order to fulfill the contract of the interface.

Ex. HTTP doesn't care what TCP does under the hood, it just cares that it can send data in the right order without losing packets.

Ex. (VC Assist) 

