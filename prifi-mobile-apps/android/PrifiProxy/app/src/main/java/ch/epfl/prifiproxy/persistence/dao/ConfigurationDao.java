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
    Configuration getConfiguration(int id);

    @Query("SELECT * FROM Configuration")
    LiveData<List<Configuration>> getAllConfigurations();

    @Query("SELECT * FROM Configuration WHERE groupId = :groupId ORDER BY priority ASC")
    LiveData<List<Configuration>> getConfigurations(int groupId);

    @Query("SELECT * FROM ConfigurationGroup ORDER BY name ASC")
    LiveData<List<ConfigurationGroup>> getAllConfigurationGroups();

    @Query("SELECT * FROM ConfigurationGroup WHERE id = :id ORDER BY name ASC")
    LiveData<List<ConfigurationGroup>> getAllConfigurationById(int id);

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    long insertConfiguration(Configuration configuration);

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    long[] insertConfigurations(List<Configuration> configuration);

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    long insertConfigurationGroup(ConfigurationGroup group);

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    long[] insertConfigurationGroups(ConfigurationGroup... groups);

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    long[] insertConfigurationGroups(List<ConfigurationGroup> groups);

    @Update
    void updateConfigurations(Configuration... configurations);

    @Update
    void updateConfigurationGroups(ConfigurationGroup... groups);

    @Update
    void updateConfigurationGroups(List<ConfigurationGroup> groups);

    @Delete
    void deleteConfigurations(Configuration... configurations);

    @Delete
    void deleteConfigurationGroups(ConfigurationGroup... groups);

    @Query("DELETE FROM Configuration WHERE groupId = :groupId")
    void deleteAllForGroupId(int groupId);

    @Query("DELETE FROM Configuration")
    void deleteAllConfigurations();

    @Query("DELETE FROM ConfigurationGroup")
    void deleteAllConfigurationGroups();
}
