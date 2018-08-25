package ch.epfl.prifiproxy.persistence.dao;

import android.arch.lifecycle.LiveData;
import android.arch.persistence.room.Dao;
import android.arch.persistence.room.Delete;
import android.arch.persistence.room.Insert;
import android.arch.persistence.room.OnConflictStrategy;
import android.arch.persistence.room.Query;
import android.arch.persistence.room.Update;

import java.util.List;

import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

@Dao
public interface ConfigurationGroupDao {
    @Query("SELECT * FROM ConfigurationGroup WHERE id = :id")
    ConfigurationGroup get(int id);

    @Query("SELECT * FROM ConfigurationGroup ORDER BY name ASC")
    LiveData<List<ConfigurationGroup>> getAll();

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    long[] insert(ConfigurationGroup... groups);

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    long[] insert(List<ConfigurationGroup> groups);

    @Update
    void update(ConfigurationGroup... groups);

    @Update
    void update(List<ConfigurationGroup> groups);

    @Delete
    void delete(ConfigurationGroup... groups);

    @Delete
    void delete(List<ConfigurationGroup> groups);

    @Query("DELETE FROM ConfigurationGroup")
    void deleteAll();
}
