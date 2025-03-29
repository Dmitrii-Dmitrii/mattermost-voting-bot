box.cfg({
    listen = 3301,
})

if not box.space.votes then
    box.schema.space.create('votes', {
        engine = 'memtx',
    })

    box.space.votes:format({
        { name = 'id', type = 'unsigned' },
        { name = 'creator_id', type = 'string' },
        { name = 'question', type = 'string' },
        { name = 'answers', type = 'array' },
        { name = 'votes', type = 'map' },
        { name = 'is_active', type = 'boolean' },
        { name = 'is_multiple_answers', type = 'boolean' },
        { name = 'is_anonymous', type = 'boolean' },
        { name = 'expires_at', type = 'unsigned' },
    })
end

local votes_space = box.space.votes

if not votes_space.index.primary then
    votes_space:create_index('primary', {
        type = 'hash',
        parts = {1, 'unsigned'}
    })
end

if not votes_space.index.is_active then
    votes_space:create_index('is_active', {
        type = 'tree',
        parts = {6, 'boolean'},
        unique = false,
    })
end

if not votes_space.index.is_multiple_answers then
    votes_space:create_index('is_multiple_answers', {
        type = 'tree',
        parts = {7, 'boolean'},
        unique = false,
    })
end

if not votes_space.index.is_anonymous then
    votes_space:create_index('is_anonymous', {
        type = 'tree',
        parts = {8, 'boolean'},
        unique = false,
    })
end

if not votes_space.index.expires_at then
    votes_space:create_index('expires_at', {
        type = 'tree',
        parts = {9, 'unsigned'},
        unique = false,
    })
end
