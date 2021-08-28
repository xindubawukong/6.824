import os

files = os.listdir('../src/raft/tmp')
for file in files:
    with open('../src/raft/tmp/' + file, 'r') as f:
        s = f.read()
        if len(s) > 0 and file.endswith('err'):
            print(file)
