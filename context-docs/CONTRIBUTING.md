# Contributing

Este repositório fornece contratos (interfaces) e adaptadores para provedores de nuvem.

Diretrizes importantes:

- Mantenha objetos pequenos e evite cópias desnecessárias para reduzir uso de GC.
- Métodos de listagem devem permitir paginação e limite de itens (`MaxItems`).
- Uploads/downloads devem ser streaming; evite buffers inteiros.
- Documente complexidade esperada (O(n)) para operações públicas.
