for f in experiment*;
do
    res=$(cat $f | grep "PCAPLog-individuals" | ./data_explore2.py)
    echo "$f $res"
done