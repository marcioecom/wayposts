# Contexto do Wayposts

Vocabulário do domínio. Este arquivo é um glossário. Ele define o que cada termo
significa, não como o sistema é construído. Decisões de arquitetura vivem em
`docs/adr/`.

## Termos

### Post
Unidade de conteúdo publicada por um User: texto e zero a quatro anexos de mídia.
Na v1 todo Post é raiz. Não existem Reply nem Repost.

### Reaction
Em v1, sinônimo de **Like**: marca idempotente de um User sobre um Post.
Não confundir com Reply (resposta, que seria um Post) nem com Repost (redistribuição).

### Follow
Relação direcionada e assimétrica entre dois Users: *follower* → *followee*.
Um Follow é o que habilita um Post do Followee a alcançar a Home Timeline do Follower.

### Home Timeline
Sequência de Posts, do mais recente para o mais antigo, formada pelos autores que um
User segue. É a leitura que o fanout serve.

### Profile Timeline
Sequência de Posts de um único autor. Não depende de Follow.

### Fanout
Distribuição de um Post recém-publicado para as Home Timelines dos seguidores do autor.
É uma operação de dados, não um mecanismo de roteamento. Não confundir com o
*fanout exchange* do AMQP, que apenas copia uma mensagem para várias filas.

### Celebridade
Autor cuja contagem de seguidores ultrapassa o threshold de fanout. Seus Posts não são
materializados nas Home Timelines; são lidos e mesclados no momento da leitura.

### Fanout lag
Intervalo entre a publicação de um Post e sua disponibilidade na Home Timeline dos
seguidores. Mede frescor, não latência de request.

### Nudge
Aviso de que existem Posts novos na Home Timeline desde a última leitura. Carrega apenas
um contador. Não confundir com entrega de Post.

## Fora de escopo na v1

Reply, Repost/Quote, busca, hashtags/trending, DM, notificações persistidas,
ranking algorítmico, moderação, apps mobile.
