package ch.epfl.prifiproxy.persistence.dao;

import android.arch.lifecycle.LiveData;
import android.arch.persistence.room.Dao;
import android.arch.persistence.room.Delete;
import android.arch.persistence.room.Insert;
import android.arch.persistence.room.OnConflictStrategy;
import android.arch.persistence.room.Query;
import android.arch.persistence.room.Update;

import java.util.List;

import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

@Dao
public interface ConfigurationDao {
    @Query("SELECT * FROM Configuration WHERE id = :id")
    LiveData<Configuration> get(int id);

    @Query("SELECT * FROM Configuration WHERE groupId = :groupId ORDER BY priority ASC")
    LiveData<List<Configuration>> getForGroup(int groupId);

    @Query("SELECT * FROM Configuration WHERE isActive = 1")
    Configuration getActive();

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    long[] insert(Configuration... configurations);

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    long[] insert(List<Configuration> configurations);

    @Update
    void update(Configuration... configurations);

    @Update
    void update(List<Configuration> configurations);

    @Delete
    void delete(Configuration... configurations);

    @Delete
    void delete(List<Configuration> configurations);

    @Query("DELETE FROM Configuration")
    void deleteAll();
}
